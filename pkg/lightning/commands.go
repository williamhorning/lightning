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

	for _, plugin := range b.plugins {
		if err := plugin.SetupCommands(b.commands); err != nil {
			err := LogError(err, "Failed to setup commands for plugin",
				map[string]any{"plugin": plugin.Name()}, nil)
			slog.Warn("lightning: commands for plugin might not be available", "err", err)
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
				_, err := bot.SendMessage(Message{
					Content: message,
					Author:  bot.author,
					BaseMessage: BaseMessage{
						ChannelID: event.ChannelID,
						Time:      time.Now(),
						Plugin:    event.Plugin,
					},
				}, nil)

				return err
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
		event.Subcommand = &(event.Options)[0]
		event.Options = (event.Options)[1:]
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
		response = LogError(err, "Error executing command", map[string]any{
			"command": command.Name,
			"event":   event.EventID,
		}, nil).Error()
	}

	if err = event.Reply(response); err != nil {
		err := LogError(err, "Error sending command response", map[string]any{
			"command": command.Name,
			"event":   event.EventID,
		}, nil)
		slog.Warn("lightning: failed to respond to command", "err", err)
	}
}

func processCommandArguments(command Command, event *CommandEvent) bool {
	for _, arg := range command.Arguments {
		if event.Arguments[arg.Name] != "" {
			continue
		}

		if len(event.Options) > 0 {
			event.Arguments[arg.Name] = (event.Options)[0]
			event.Options = (event.Options)[1:]
		}

		if event.Arguments[arg.Name] != "" || !arg.Required {
			continue
		}

		err := event.Reply(
			"Please provide the " + arg.Name + " argument. Try using the `" + event.Prefix + "help` command.",
		)
		if err != nil {
			err := LogError(err, "Error sending missing argument response",
				map[string]any{"argument": arg.Name, "command": command.Name, "event": event.EventID}, nil)
			slog.Warn("lightning: failed to respond to command", "err", err)
		}

		return false
	}

	return true
}
