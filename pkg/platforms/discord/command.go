package discord

import (
	"log"
	"strconv"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
)

func lightningToDiscordCommands(original map[string]lightning.Command) []applicationCommand {
	cmds := make([]applicationCommand, 0, len(original))

	for _, cmd := range original {
		cmds = append(cmds, applicationCommand{
			Type:        commandTypeChatInput,
			Name:        cmd.Name,
			Description: cmd.Description,
			Options:     lightningToDiscordCommandOptions(&cmd),
		})
	}

	return cmds
}

func lightningToDiscordCommandOptions(cmd *lightning.Command) []commandOption {
	var options []commandOption

	if len(cmd.Subcommands) == 0 {
		for _, arg := range cmd.Arguments {
			options = append(options, commandOption{
				Type:        optString,
				Name:        arg.Name,
				Description: arg.Description,
				Required:    true,
			})
		}
	}

	for _, sub := range cmd.Subcommands {
		options = append(options, commandOption{
			Type:        optSubCommand,
			Name:        sub.Name,
			Description: sub.Description,
			Options:     lightningToDiscordCommandOptions(&sub),
		})
	}

	return options
}

func discordToLightningCommand(
	client *client,
	interaction *interactionCreateEvent,
) *lightning.CommandEvent {
	if interaction.Type != interactionApplicationCommand {
		return nil
	}

	args := make(map[string]string)

	var subcommand *string

	for _, option := range interaction.Data.Options {
		switch option.Type {
		case optString:
			args[option.Name] = option.Value
		case optSubCommand:
			val := option.Name
			subcommand = &val

			for _, subopt := range option.Options {
				args[subopt.Name] = subopt.Value
			}
		default:
		}
	}

	timestamp := time.Now()
	if id, err := strconv.ParseInt(string(interaction.ID), 10, 64); err == nil {
		timestamp = time.UnixMilli((id >> 22) + 1420070400000)
	}

	return &lightning.CommandEvent{
		CommandOptions: &lightning.CommandOptions{
			Arguments: args, Author: discordToLightningAuthor(
				client, interaction.getUser(), interaction.Member, interaction.GuildID,
			), BaseMessage: lightning.BaseMessage{
				EventID:   string(interaction.ID),
				ChannelID: string(*interaction.ChannelID), Time: timestamp,
			}, Prefix: "/",
			Reply: func(message *lightning.Message, sensitive bool) {
				message.BaseMessage = lightning.BaseMessage{Time: time.Now(), ChannelID: string(*interaction.ChannelID)}
				msg := lightningToDiscordSendable(client, message, &lightning.SendOptions{})

				if sensitive {
					msg.Flags = messageFlagsEphemeral
				}

				if err := client.respondInteraction(interaction.ID, interaction.Token, &interactionResponse{
					Type: respChannelMessageWithSource, Data: msg.toInteraction(),
				}); err != nil {
					log.Printf("%s: failed responding to command: %v\n", client.product, err)
				}
			},
		},
		Command: interaction.Data.Name, Subcommand: subcommand,
	}
}
