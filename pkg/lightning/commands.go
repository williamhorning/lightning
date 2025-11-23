package lightning

import "strings"

// AddCommand takes [Command]s and registers it with the built-in
// text command handler and platform-specific command systems.
func (b *Bot) AddCommand(commands ...Command) {
	for _, command := range commands {
		b.commands[command.Name] = &command
	}

	for _, plugin := range b.plugins {
		_ = plugin.SetupCommands(b.commands)
	}
}

func handleMessageCommand(bot *Bot, event *Message) {
	if len(event.Content) <= len(bot.prefix) || event.Content[:len(bot.prefix)] != bot.prefix {
		return
	}

	args := strings.Fields(event.Content[len(bot.prefix):])
	if len(args) == 0 {
		args = []string{"help"}
	}

	commandName := args[0]
	options := args[1:]

	reply := func(msg *Message, sensitive bool) {
		plugin, channel, ok := bot.getPluginFromChannel(event.ChannelID)
		if !ok {
			return
		}

		msg.ChannelID = channel
		msg.RepliedTo = append(msg.RepliedTo, event.EventID)

		if sensitive {
			_, _ = plugin.SendCommandResponse(msg, nil, event.Author.ID)
		} else {
			_, _ = plugin.SendMessage(msg, nil)
		}
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

	if len(command.Subcommands) != 0 && len(event.Options) != 0 && event.Subcommand == nil {
		event.Subcommand = &event.Options[0]
		event.Options = event.Options[1:]
	}

	if event.Subcommand != nil {
		if cmd, ok := command.Subcommands[*event.Subcommand]; ok {
			command = &cmd
		}
	}

	for _, arg := range command.Arguments {
		if event.Arguments[arg.Name] == "" && len(event.Options) > 0 {
			event.Arguments[arg.Name] = event.Options[0]
			event.Options = event.Options[1:]
		}
	}

	command.Executor(event.CommandOptions)
}
