package lightning

import (
	"log/slog"
	"strings"
)

// AddCommand takes a [Command] and registers it with the built-in
// text command handler and any platform-specific command systems.
func (b *Bot) AddCommand(command *Command) {
	b.commands[command.Name] = command

	for _, plugin := range b.plugins {
		if err := plugin.SetupCommands(b.commands); err != nil {
			slog.Error("lightning: failed to setup commands for plugin", "err",
				PluginMethodError{err, "", "AddCommand", "one or more plugins failed to register command"})
		}
	}
}

func handleMessageCommand(bot *Bot, event *Message) {
	if !strings.HasPrefix(event.Content, bot.prefix) {
		return
	}

	args := strings.Fields(strings.TrimPrefix(event.Content, bot.prefix))
	if len(args) == 0 {
		args = []string{"help"}
	}

	commandName := args[0]
	options := args[1:]

	handleCommandEvent(bot, &CommandEvent{
		CommandOptions: CommandOptions{&event.BaseMessage, make(map[string]string), bot, bot.prefix},
		Command:        commandName,
		Options:        options,
		Reply: func(message string, sensitive bool) error {
			plugin, channel, ok := bot.getPluginFromChannel(event.ChannelID)
			if !ok {
				return MissingPluginError{}
			}

			msg := &Message{BaseMessage: BaseMessage{ChannelID: channel}, Author: bot.author, Content: message}

			var err error

			if sensitive {
				_, err = plugin.SendCommandResponse(msg, nil, event.Author.ID)
			} else {
				_, err = plugin.SendMessage(msg, nil)
			}

			if err == nil {
				return nil
			}

			return PluginMethodError{err, event.ChannelID, "CommandReply", "failed to send command response"}
		},
	})
}

func handleCommandEvent(bot *Bot, event *CommandEvent) {
	event.Bot = bot

	command, exists := bot.commands[event.Command]
	if !exists {
		command = bot.commands["help"]
	}

	for _, arg := range event.Options {
		if len(command.Subcommands) > 0 && event.Subcommand == nil {
			event.Subcommand = &arg
			event.Options = event.Options[1:]
		}
	}

	for _, subcommand := range command.Subcommands {
		if event.Subcommand != nil && subcommand.Name == *event.Subcommand {
			command = subcommand

			break
		}
	}

	handleCommandOptions(command, event)

	if err := event.Reply(command.Executor(event.CommandOptions), command.Sensitive); err != nil {
		slog.Warn("lightning: failed to respond to command", "err",
			PluginMethodError{err, event.ChannelID, "eventReply", "failed to reply to command event"})
	}
}

func handleCommandOptions(command *Command, event *CommandEvent) {
	for _, arg := range command.Arguments {
		if event.Arguments[arg.Name] != "" {
			continue
		}

		if len(event.Options) > 0 {
			event.Arguments[arg.Name] = event.Options[0]
			event.Options = event.Options[1:]
		}
	}
}
