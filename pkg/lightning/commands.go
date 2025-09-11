package lightning

import (
	"strings"
)

// AddCommand takes [Command]s and registers it with the built-in
// text command handler and any platform-specific command systems.
func (b *Bot) AddCommand(commands ...*Command) error {
	var errs []error

	for _, command := range commands {
		if command == nil {
			continue
		}

		b.commands[command.Name] = command
	}

	for _, plugin := range b.plugins {
		if err := plugin.SetupCommands(b.commands); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return &PluginMethodError{"", "AddCommand", "failed to register command", errs}
	}

	return nil
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

	reply := func(msg *Message, sensitive bool) error {
		plugin, channel, ok := bot.getPluginFromChannel(event.ChannelID)
		if !ok {
			return MissingPluginError{}
		}

		msg.ChannelID = channel

		var err error

		if sensitive {
			_, err = plugin.SendCommandResponse(msg, nil, event.Author.ID)
		} else {
			_, err = plugin.SendMessage(msg, nil)
		}

		if err == nil {
			return nil
		}

		return &PluginMethodError{event.ChannelID, "CommandReply", "failed to send command response", []error{err}}
	}

	handleCommandEvent(bot, &CommandEvent{
		CommandOptions: &CommandOptions{&event.BaseMessage, make(map[string]string), bot, reply, bot.prefix},
		Command:        commandName,
		Options:        options,
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

	command.Executor(event.CommandOptions)
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
