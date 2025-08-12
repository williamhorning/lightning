package lightning

import (
	"log/slog"
	"strings"
	"time"
)

// AddCommand takes a [Command] and registers it with the built-in
// text command handler and any platform-specific command systems.
func (b *Bot) AddCommand(command Command) {
	b.commands[command.Name] = command

	for pluginName, plugin := range b.plugins {
		if err := plugin.SetupCommands(b.commands); err != nil {
			methodErr := PluginMethodError{err, pluginName, "SetupCommands", "failed to setup commands"}
			slog.Warn("lightning: commands for plugin might not be available", "err", methodErr)
		}
	}
}

func handleMessageCommand(prefix string) func(bot *Bot, event *Message) {
	return func(bot *Bot, event *Message) {
		if !strings.HasPrefix(event.Content, prefix) {
			return
		}

		content := strings.TrimPrefix(event.Content, prefix)

		args := strings.Fields(content)
		if len(args) == 0 {
			args = []string{"help"}
		}

		commandName := args[0]
		options := args[1:]

		handleCommandEvent(bot, &CommandEvent{
			CommandOptions: CommandOptions{
				Arguments:   make(map[string]string),
				BaseMessage: event.BaseMessage,
				Prefix:      prefix,
			},
			Command: commandName,
			Options: options,
			Reply: func(message string) error {
				plugin, _, ok := bot.getPluginFromChannel(event.ChannelID)
				if !ok {
					return MissingPluginError{}
				}

				_, err := plugin.SendCommandResponse(Message{
					Content: message,
					Author:  bot.author,
					BaseMessage: BaseMessage{
						ChannelID: event.ChannelID,
						Time:      time.Now(),
					},
				}, nil, event.Author.ID)

				if err == nil {
					return nil
				}

				return PluginMethodError{err, event.ChannelID, "SendCommandResponse", "failed to send command response"}
			},
		})
	}
}

func handleCommandEvent(bot *Bot, event *CommandEvent) {
	event.Bot = bot

	command, exists := bot.commands[event.Command]
	if !exists {
		command = bot.commands["help"]
	}

	if len(event.Options) != 0 {
		event.Subcommand = &event.Options[0]
		event.Options = event.Options[1:]
	}

	for _, subcommand := range command.Subcommands {
		if event.Subcommand != nil && subcommand.Name == *event.Subcommand {
			command = subcommand

			break
		}
	}

	if !processCommandArguments(command, event) {
		return
	}

	response, err := command.Executor(event.CommandOptions)
	if err != nil {
		response = "Something went wrong when running that command!\nPlease try again later."
	}

	if err = event.Reply(response); err != nil {
		err := PluginMethodError{err, event.ChannelID, "eventReply", "failed to reply to command event"}
		slog.Warn("lightning: failed to respond to command", "err", err)
	}
}

func processCommandArguments(command Command, event *CommandEvent) bool {
	for _, arg := range command.Arguments {
		if event.Arguments[arg.Name] != "" {
			continue
		}

		if len(event.Options) > 0 {
			event.Arguments[arg.Name] = event.Options[0]
			event.Options = event.Options[1:]
		}

		if event.Arguments[arg.Name] != "" || !arg.Required {
			continue
		}

		err := event.Reply(
			"Please provide the " + arg.Name + " argument. Try using the `" + event.Prefix + "help` command.",
		)
		if err != nil {
			err := PluginMethodError{err, event.ChannelID, "eventReply", "failed to reply to command event"}
			slog.Warn("lightning: failed to respond to command", "err", err)
		}

		return false
	}

	return true
}
