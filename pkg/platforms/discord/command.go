package discord

import (
	"log"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func lightningToDiscordCommands(original map[string]lightning.Command) []discord.ApplicationCommandCreate {
	var cmds []discord.ApplicationCommandCreate //nolint:prealloc

	for _, cmd := range original {
		cmds = append(cmds, discord.SlashCommandCreate{
			Name:        cmd.Name,
			Description: cmd.Description,
			Options:     lightningToDiscordCommandOptions(&cmd),
		})
	}

	return cmds
}

func lightningToDiscordCommandOptions(cmd *lightning.Command) []discord.ApplicationCommandOption {
	var options []discord.ApplicationCommandOption //nolint:prealloc

	if len(cmd.Subcommands) == 0 {
		for _, arg := range cmd.Arguments {
			options = append(options, discord.ApplicationCommandOptionString{
				Name:        arg.Name,
				Description: arg.Description,
				Required:    true,
			})
		}
	}

	for _, sub := range cmd.Subcommands {
		options = append(options, discord.ApplicationCommandOptionSubCommand{
			Name:        sub.Name,
			Description: sub.Description,
			Options:     lightningToDiscordCommandOptions(&sub),
		})
	}

	return options
}

func discordToLightningCommand(
	session *bot.Client,
	interaction *events.ApplicationCommandInteractionCreate,
	cdn string,
) *lightning.CommandEvent {
	if interaction.Type() != discord.InteractionTypeApplicationCommand {
		return nil
	}

	args := make(map[string]string)
	data := interaction.SlashCommandInteractionData()

	for _, option := range data.Options {
		if option.Type == discord.ApplicationCommandOptionTypeString {
			args[option.Name] = string(option.Value)
		}
	}

	return &lightning.CommandEvent{
		CommandOptions: &lightning.CommandOptions{
			Arguments: args, BaseMessage: lightning.BaseMessage{
				EventID:   interaction.ID().String(),
				ChannelID: interaction.Channel().String(), Time: interaction.CreatedAt(),
			}, Prefix: "/",
			Reply: func(message *lightning.Message, sensitive bool) {
				message.BaseMessage = lightning.BaseMessage{Time: time.Now(), ChannelID: interaction.Channel().String()}
				msg := lightningToDiscordSendable(session, message, &lightning.SendOptions{}, cdn)

				defer func() {
					for _, cancel := range msg.cancels {
						cancel()
					}
				}()

				if sensitive {
					msg.Flags = discord.MessageFlagEphemeral
				}

				if err := interaction.Respond(discord.InteractionResponseTypeCreateMessage, msg); err != nil {
					log.Printf("discord: failed responding to command: %v\n", err)
				}
			},
		},
		Command: data.CommandName(), Subcommand: data.SubCommandName,
	}
}
