// Package discord provides a [lightning.Plugin] implementation for Discord.
// To use Discord support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("discord", discord.New)
//
//	bot.UsePluginType("discord", "", map[string]string{
//		// ...
//	})
package discord

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"codeberg.org/jersey/lightning/internal/cache"
	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

// New creates a new [lightning.Plugin] that provides Discord support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]string{
//		"api_host": "", // optional, can specify a non-Discord API implementation. must be paired with cdn_host
//		"cdn_host": "", // optional, can specify a non-Discord CDN implementation. must be paired with api_host
//		"token": "",    // required, a string with your Discord bot token
//	}
//
// Note that you MUST enable the Message Content intent for the plugin to work.
func New(cfg map[string]string) (lightning.Plugin, error) {
	apiHost := "https://discord.com/api/"

	if host, ok := cfg["api_host"]; ok {
		apiHost = "https://" + host + "/api"

		rest.Version = 9
	}

	cdnHost := "cdn.discordapp.com"

	if host, ok := cfg["cdn_host"]; ok {
		cdnHost = host
	}

	session, err := disgo.New(
		cfg["token"],
		bot.WithRestClientConfigOpts(rest.WithURL(apiHost)),
		bot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentsNonPrivileged, gateway.IntentMessageContent)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	if err = session.OpenGateway(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to open Discord session: %w", err)
	}

	invite := "https://discord.com/oauth2/authorize?client_id=" + session.ApplicationID.String() +
		"&permissions=8&scope=bot"

	if apiHost != "https://discord.com/api/" {
		invite = strings.ReplaceAll(invite, "discord.com", "fermi.chat")
	}

	session.AddEventListeners(bot.NewListenerFunc(func(e *events.Ready) {
		log.Printf("discord: ready as %s in %d servers! %s\n", e.User.Username, len(e.Guilds), invite)
	}))

	return &discordPlugin{session: session, cdnHost: cdnHost}, nil
}

type discordPlugin struct {
	session  *bot.Client
	webhooks cache.Expiring[snowflake.ID, bool]
	cdnHost  string
}

func (p *discordPlugin) SetupChannel(channel string) (map[string]string, error) {
	id, err := snowflake.Parse(channel)
	if err != nil {
		return nil, &snowflakeError{channel, true}
	}

	wh, err := p.session.Rest.CreateWebhook(id, discord.WebhookCreate{Name: channel})
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook for channel: %w", err)
	}

	return map[string]string{"id": wh.ID().String(), "token": wh.Token}, nil
}

func (p *discordPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	channelID, err := p.getOutgoingChannel(message, opts)
	if err != nil {
		return nil, err
	}

	msg := lightningToDiscordSendable(p.session, message, opts, p.cdnHost)

	defer func() {
		for _, cancel := range msg.cancels {
			cancel()
		}
	}()

	chunks := slices.Collect(slices.Chunk([]rune(msg.Content), 2000))

	if len(chunks) == 0 {
		chunks = [][]rune{nil}
	}

	ids := make([]string, 0, len(chunks))

	for idx, chunk := range chunks {
		if idx != 0 {
			msg.Files = nil
			msg.Embeds = nil
		}

		msg.Content = string(chunk)

		var res *discord.Message

		var err error

		if opts.ChannelData == nil {
			res, err = p.session.Rest.CreateMessage(channelID, msg.MessageCreate)
		} else {
			res, err = p.session.Rest.CreateWebhookMessage(
				channelID, opts.ChannelData["token"], msg.toWebhook(), rest.CreateWebhookMessageParams{Wait: true},
			)
		}

		if err != nil {
			return ids, getError(err, "send")
		}

		ids = append(ids, res.ID.String())
	}

	return ids, nil
}

func (p *discordPlugin) EditMessage(
	message *lightning.Message, ids []string, opts *lightning.SendOptions,
) ([]string, error) {
	channelID, err := p.getOutgoingChannel(message, opts)
	if err != nil {
		return nil, err
	}

	message.Attachments = nil

	msg := lightningToDiscordSendable(p.session, message, opts, p.cdnHost)

	defer func() {
		for _, cancel := range msg.cancels {
			cancel()
		}
	}()

	chunks := slices.Collect(slices.Chunk([]rune(msg.Content), 2000))

	if len(chunks) == 0 {
		chunks = [][]rune{nil}
	}

	for idx, chunk := range chunks {
		if idx != 0 {
			msg.Embeds = nil
		}

		msg.Content = string(chunk)

		var res *discord.Message

		var err error

		msgID, err := snowflake.Parse(ids[idx])
		if err != nil {
			return ids, &snowflakeError{ids[idx], false}
		}

		if opts.ChannelData == nil {
			res, err = p.session.Rest.UpdateMessage(channelID, msgID, msg.toEdit())
		} else {
			res, err = p.session.Rest.UpdateWebhookMessage(
				channelID, opts.ChannelData["token"], msgID, msg.toWebhookEdit(), rest.UpdateWebhookMessageParams{},
			)
		}

		if err != nil {
			return ids, getError(err, "send")
		}

		ids = append(ids, res.ID.String())
	}

	return ids, nil
}

func (p *discordPlugin) DeleteMessage(channel string, ids []string) error {
	channelID, err := snowflake.Parse(channel)
	if err != nil {
		return &snowflakeError{channel, true}
	}

	snowflakeIDs := make([]snowflake.ID, 0, len(ids))

	for _, idStr := range ids {
		idInt, err := snowflake.Parse(idStr)
		if err != nil {
			return &snowflakeError{channel, false}
		}

		snowflakeIDs = append(snowflakeIDs, idInt)
	}

	if len(snowflakeIDs) == 1 {
		return getError(p.session.Rest.DeleteMessage(channelID, snowflakeIDs[0]), "delete")
	}

	return getError(p.session.Rest.BulkDeleteMessages(channelID, snowflakeIDs), "delete (bulk)")
}

func (p *discordPlugin) SetupCommands(command map[string]lightning.Command) {
	_, err := p.session.Rest.SetGlobalCommands(p.session.ApplicationID, lightningToDiscordCommands(command))
	if err != nil {
		log.Printf("discord: failed to setup commands: %v\n", err)
	}
}

func (p *discordPlugin) ListenMessages() <-chan *lightning.Message {
	channel := make(chan *lightning.Message, 1000)

	p.session.AddEventListeners(bot.NewListenerFunc(func(m *events.MessageCreate) {
		if msg := discordToLightning(&p.webhooks, p.session, &m.Message, p.cdnHost); msg != nil {
			channel <- msg
		}
	}))

	return channel
}

func (p *discordPlugin) ListenEdits() <-chan *lightning.EditedMessage {
	channel := make(chan *lightning.EditedMessage, 1000)

	p.session.AddEventListeners(bot.NewListenerFunc(func(message *events.MessageUpdate) {
		if msg := discordToLightning(&p.webhooks, p.session, &message.Message, p.cdnHost); msg != nil {
			if message.Message.EditedTimestamp == nil {
				now := time.Now()
				message.Message.EditedTimestamp = &now
			}

			channel <- &lightning.EditedMessage{Message: msg, Edited: *message.Message.EditedTimestamp}
		}
	}))

	return channel
}

func (p *discordPlugin) ListenDeletes() <-chan *lightning.BaseMessage {
	channel := make(chan *lightning.BaseMessage, 1000)

	p.session.AddEventListeners(bot.NewListenerFunc(func(m *events.MessageDelete) {
		channel <- &lightning.BaseMessage{
			EventID: m.MessageID.String(), ChannelID: m.ChannelID.String(), Time: time.Now(),
		}
	}))

	return channel
}

func (p *discordPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	channel := make(chan *lightning.CommandEvent, 1000)

	p.session.AddEventListeners(bot.NewListenerFunc(func(m *events.ApplicationCommandInteractionCreate) {
		if cmd := discordToLightningCommand(p.session, m, p.cdnHost); cmd != nil {
			channel <- cmd
		}
	}))

	return channel
}
