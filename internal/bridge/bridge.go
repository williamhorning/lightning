// Package bridge implements a bridge bot based on Lightning, the framework, for Lightning, the bot.
package bridge

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func handleBridgeMessage(bot *lightning.Bot, database data.Database, event data.EventType, dat any) error {
	base := extractBase(dat)

	bridge, priorMsg, err := resolveBridgeData(database, base, event)
	if err != nil {
		return fmt.Errorf("failed to get bridge for (%s in %s): %w", base.EventID, base.ChannelID, err)
	}

	if bridge == nil || bridge.ID == "" || bridge.GetChannelDisabled(base.ChannelID).Read {
		return nil
	}

	repliedTo := getRepliedToMessage(database, dat)
	messages := processMessages(bot, database, bridge, event, base, dat, repliedTo, priorMsg)

	return updateDatabase(database, event, base, bridge, messages)
}

func resolveBridgeData(
	database data.Database,
	base lightning.BaseMessage,
	event data.EventType,
) (*data.Bridge, *data.BridgeMessageCollection, error) {
	switch event {
	case data.TypeCreate:
		bridge, err := database.GetBridgeByChannel(base.ChannelID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get bridge from database: %w", err)
		}

		return &bridge, nil, nil
	case data.TypeEdit, data.TypeDelete:
		prior, err := database.GetMessage(base.EventID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get message from database: %w", err)
		}

		if event == data.TypeEdit && prior.ID != base.EventID {
			return nil, nil, nil
		}

		bridge, err := database.GetBridge(prior.BridgeID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get bridge from database: %w", err)
		}

		return &bridge, &prior, nil
	default:
		return nil, nil, nil
	}
}

func extractBase(dat any) lightning.BaseMessage {
	switch evt := dat.(type) {
	case lightning.EditedMessage:
		return evt.Message.BaseMessage
	case lightning.Message:
		return evt.BaseMessage
	case lightning.BaseMessage:
		return evt
	default:
		return lightning.BaseMessage{}
	}
}

func extractMessage(dat any) lightning.Message {
	switch v := dat.(type) {
	case lightning.EditedMessage:
		return *v.Message
	case lightning.Message:
		return v
	default:
		return lightning.Message{}
	}
}

func getRepliedToMessage(database data.Database, dat any) *data.BridgeMessageCollection {
	msg := extractMessage(dat)

	if len(msg.RepliedTo) == 0 {
		return nil
	}

	repliedTo, err := database.GetMessage(msg.RepliedTo[0])
	if err != nil {
		log.Printf("bridge: failed to get replied_to for %s: %v\n", msg.RepliedTo[0], err)

		return nil
	}

	return &repliedTo
}

func updateDatabase(
	database data.Database,
	event data.EventType,
	base lightning.BaseMessage,
	bridge *data.Bridge,
	messages []data.ChannelMessage,
) error {
	switch event {
	case data.TypeCreate, data.TypeEdit:
		err := database.CreateMessage(data.BridgeMessageCollection{
			ID:       base.EventID,
			BridgeID: bridge.ID,
			Messages: messages,
		})
		if err != nil {
			return fmt.Errorf("updateDatabase failed: %w", err)
		}
	case data.TypeDelete:
		err := database.DeleteMessage(base.EventID)
		if err != nil {
			return fmt.Errorf("updateDatabase failed: %w", err)
		}
	default:
	}

	return nil
}

func processMessages(
	bot *lightning.Bot,
	database data.Database,
	bridge *data.Bridge,
	event data.EventType,
	base lightning.BaseMessage,
	dat any,
	repliedTo *data.BridgeMessageCollection,
	priorMsg *data.BridgeMessageCollection,
) []data.ChannelMessage {
	messages := make([]data.ChannelMessage, 0, len(bridge.Channels)+1)
	results := make(chan *data.ChannelMessage, len(bridge.Channels))
	wait := sync.WaitGroup{}

	for _, channel := range bridge.Channels {
		if channel.ID == base.ChannelID || channel.Disabled.Write {
			continue
		}

		wait.Go(func() {
			priorIDs := priorMsg.GetChannelMessageIDs(channel.ID)
			if event != data.TypeCreate && len(priorIDs) == 0 {
				return
			}

			message := handleChannel(bot, database, bridge, &channel, event, dat, repliedTo, priorIDs)
			if message != nil {
				results <- message
			}
		})
	}

	wait.Wait()
	close(results)

	for msg := range results {
		messages = append(messages, *msg)
	}

	messages = append(messages, data.ChannelMessage{
		ChannelID:  base.ChannelID,
		MessageIDs: []string{base.EventID},
	})

	return messages
}

func handleChannel(
	bot *lightning.Bot,
	database data.Database,
	bridge *data.Bridge,
	channel *data.BridgeChannel,
	event data.EventType,
	dat any,
	repliedTo *data.BridgeMessageCollection,
	priorIDs []string,
) *data.ChannelMessage {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("bridge: panic in channel %s: %#+v", channel.ID, r)
		}
	}()

	opts := &lightning.SendOptions{
		AllowEveryonePings: bridge.Settings.AllowEveryone,
		ChannelData:        channel.Data.(map[string]string),
	}

	var err error

	resultIDs := priorIDs

	switch event {
	case data.TypeCreate, data.TypeEdit:
		msg := extractMessage(dat)
		msg.ChannelID = channel.ID
		msg.RepliedTo = repliedTo.GetChannelMessageIDs(channel.ID)

		if event == data.TypeCreate {
			resultIDs, err = bot.SendMessage(&msg, opts)
		} else if len(priorIDs) != 0 {
			err = bot.EditMessage(&msg, priorIDs, opts)
		}

	case data.TypeDelete:
		err = bot.DeleteMessages(channel.ID, priorIDs)
	default:
	}

	if err != nil {
		handleError(database, err, channel, bridge, event)

		return nil
	}

	return &data.ChannelMessage{ChannelID: channel.ID, MessageIDs: resultIDs}
}

func handleError(
	database data.Database,
	err error,
	channel *data.BridgeChannel,
	bridge *data.Bridge,
	event data.EventType,
) {
	var disabled lightning.ChannelDisabled

	disabler := new(lightning.ChannelDisabler)
	if errors.As(err, disabler) {
		if result := (*disabler).Disable(); result != nil {
			disabled = *result
		}
	}

	log.Printf("bridge: error in channel %s in bridge %s on %s: %v\n", channel.ID, bridge.ID, event, err)

	if !disabled.Read && !disabled.Write {
		return
	}

	for i, ch := range bridge.Channels {
		if ch.ID == channel.ID {
			bridge.Channels[i].Disabled = disabled

			break
		}
	}

	log.Printf("bridge: disabling channel %s in bridge %s on %s\n\tdisable: %#+v\n",
		bridge.ID, channel.ID, event, disabled)

	if err := database.CreateBridge(*bridge); err != nil {
		log.Printf("bridge: failed to disable %s in bridge %s: %v\n", channel.ID, bridge.ID, err)
	}
}
