package discord

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func lightningToDiscordCommands(original map[string]*lightning.Command) []*discordgo.ApplicationCommand {
	cmds := make([]*discordgo.ApplicationCommand, 0, len(original))

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
	options := make([]*discordgo.ApplicationCommandOption, 0, len(cmd.Arguments)+len(cmd.Subcommands))

	for _, arg := range cmd.Arguments {
		options = append(options, &discordgo.ApplicationCommandOption{
			Name:        arg.Name,
			Description: arg.Description,
			Required:    arg.Required,
			Type:        discordgo.ApplicationCommandOptionString,
		})
	}

	for _, sub := range cmd.Subcommands {
		options = append(options, &discordgo.ApplicationCommandOption{
			Name:        sub.Name,
			Description: sub.Description,
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Options:     lightningToDiscordCommandOptions(sub),
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
			Arguments: args,
			BaseMessage: &lightning.BaseMessage{
				EventID: interaction.ID, ChannelID: interaction.ChannelID, Time: timestamp,
			},
			Prefix: "/",
			Reply: func(message *lightning.Message, sensitive bool) error {
				msgs := lightningToDiscordSendable(session, message, nil)

				data := msgs[0].toInteractionResponseData()

				if sensitive {
					data.Flags = discordgo.MessageFlagsEphemeral
				}

				if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource, Data: data,
				}); err != nil {
					return fmt.Errorf("failed to respond to Discord interaction: %w", err)
				}

				return nil
			},
		},
		Command:    data.Name,
		Subcommand: subcommand,
	}
}
