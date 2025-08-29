package discord

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func getDiscordCommandOptions(arguments *lightning.Command) []*discordgo.ApplicationCommandOption {
	options := make([]*discordgo.ApplicationCommandOption, 0)

	for _, arg := range arguments.Arguments {
		options = append(options, &discordgo.ApplicationCommandOption{
			Name:        arg.Name,
			Description: arg.Description,
			Required:    arg.Required,
			Type:        discordgo.ApplicationCommandOptionString,
		})
	}

	for _, subcommand := range arguments.Subcommands {
		options = append(options, &discordgo.ApplicationCommandOption{
			Name:        subcommand.Name,
			Description: subcommand.Description,
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Options:     getDiscordCommandOptions(subcommand),
		})
	}

	return options
}

func getDiscordCommand(command map[string]*lightning.Command) []*discordgo.ApplicationCommand {
	commands := make([]*discordgo.ApplicationCommand, 0)

	for _, cmd := range command {
		commands = append(commands, &discordgo.ApplicationCommand{
			Name:        cmd.Name,
			Type:        discordgo.ChatApplicationCommand,
			Description: cmd.Description,
			Options:     getDiscordCommandOptions(cmd),
		})
	}

	return commands
}

func getLightningCommand(session *discordgo.Session, interaction *discordgo.InteractionCreate) *lightning.CommandEvent {
	if interaction.Type != discordgo.InteractionApplicationCommand {
		return nil
	}

	args := make(map[string]string)
	data := interaction.ApplicationCommandData()

	var subcommand *string

	for _, option := range data.Options {
		switch option.Type { //nolint:exhaustive // we only have string and subcommand options
		case discordgo.ApplicationCommandOptionSubCommand:
			subcommand = &option.Name

			for _, subOption := range option.Options {
				if subOption.Type == discordgo.ApplicationCommandOptionString {
					args[subOption.Name] = subOption.StringValue()
				}
			}
		case discordgo.ApplicationCommandOptionString:
			args[option.Name] = option.StringValue()
		default:
		}
	}

	timestamp, err := discordgo.SnowflakeTimestamp(interaction.ID)
	if err != nil {
		slog.Warn(fmt.Errorf("discord: failed to parse interaction timestamp: %w", err).Error(), "id", interaction.ID)
		slog.Warn("discord: using current time as fallback for interaction timestamp")

		timestamp = time.Now()
	}

	return &lightning.CommandEvent{
		CommandOptions: lightning.CommandOptions{
			Arguments: args,
			BaseMessage: &lightning.BaseMessage{
				EventID:   interaction.ID,
				ChannelID: interaction.ChannelID,
				Time:      &timestamp,
			},
			Prefix: "/",
		},
		Command:    data.Name,
		Subcommand: subcommand,
		Reply: func(message string, sensitive bool) error {
			flags := discordgo.MessageFlags(0)

			if sensitive {
				flags = discordgo.MessageFlagsEphemeral
			}

			return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: message, Flags: flags},
			})
		},
	}
}
