package discord

import (
	"log"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/bwmarrin/discordgo"
)

func lightningToDiscordCommands(original map[string]*lightning.Command) []*discordgo.ApplicationCommand {
	var cmds []*discordgo.ApplicationCommand //nolint:prealloc

	for _, cmd := range original {
		cmds = append(cmds, &discordgo.ApplicationCommand{
			Name:        cmd.Name,
			Type:        discordgo.ChatApplicationCommand,
			Description: cmd.Description,
			Options:     lightningToDiscordCommandOptions(cmd),
		})
	}

	return cmds
}

func lightningToDiscordCommandOptions(cmd *lightning.Command) []*discordgo.ApplicationCommandOption {
	var options []*discordgo.ApplicationCommandOption //nolint:prealloc

	if len(cmd.Subcommands) == 0 {
		for _, arg := range cmd.Arguments {
			options = append(options, &discordgo.ApplicationCommandOption{
				Name:        arg.Name,
				Description: arg.Description,
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			})
		}
	}

	for _, sub := range cmd.Subcommands {
		options = append(options, &discordgo.ApplicationCommandOption{
			Name:        sub.Name,
			Description: sub.Description,
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Options:     lightningToDiscordCommandOptions(&sub),
		})
	}

	return options
}

func discordToLightningCommand(
	session *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) *lightning.CommandEvent {
	if interaction.Type != discordgo.InteractionApplicationCommand {
		return nil
	}

	args := make(map[string]string)
	data := interaction.ApplicationCommandData()

	var subcommand *string

	for _, option := range data.Options {
		if option.Type == discordgo.ApplicationCommandOptionSubCommand {
			subcommand = &option.Name

			for _, subOption := range option.Options {
				if subOption.Type == discordgo.ApplicationCommandOptionString {
					args[subOption.Name] = subOption.StringValue()
				}
			}
		} else {
			args[option.Name] = option.StringValue()
		}
	}

	timestamp, err := discordgo.SnowflakeTimestamp(interaction.ID)
	if err != nil {
		timestamp = time.Now()
	}

	return &lightning.CommandEvent{
		CommandOptions: &lightning.CommandOptions{
			Arguments: args, BaseMessage: lightning.BaseMessage{
				EventID: interaction.ID, ChannelID: interaction.ChannelID, Time: timestamp,
			}, Prefix: "/",
			Reply: func(message *lightning.Message, sensitive bool) {
				message.BaseMessage = lightning.BaseMessage{Time: time.Now(), ChannelID: interaction.ChannelID}
				msg := lightningToDiscordSendable(session, message, &lightning.SendOptions{})
				data := msg.toInteractionResponseData()

				defer func() {
					for _, cancel := range msg.cancels {
						cancel()
					}
				}()

				if sensitive {
					data.Flags = discordgo.MessageFlagsEphemeral
				}

				if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource, Data: data,
				}); err != nil {
					log.Printf("discord: failed responding to command: %v\n", err)
				}
			},
		},
		Command: data.Name, Subcommand: subcommand,
	}
}
