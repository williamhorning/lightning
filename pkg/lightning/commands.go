package lightning

import (
	"log"
	"strings"
	"time"
)

// AddCommand takes [Command]s and registers it with the built-in
// text command handler and platform-specific command systems.
func (b *Bot) AddCommand(commands ...Command) {
	for _, command := range commands {
		b.commands[command.Name] = command
	}

	b.mutex.RLock()
	defer b.mutex.RUnlock()

	for _, plugin := range b.plugins {
		plugin.SetupCommands(b.commands)
	}
}

func handleTextCommand(bot *Bot, event *Message) {
	if len(event.Content) <= len(bot.prefix) || event.Content[:len(bot.prefix)] != bot.prefix {
		return
	}

	args := strings.Fields(event.Content[len(bot.prefix):])
	if len(args) == 0 {
		args = []string{"help"}
	}

	reply := func(msg *Message, sensitive bool) {
		plugin, channel, err := bot.getPluginFromChannel(event.ChannelID)
		if err != nil {
			log.Printf("lightning: failed to respond to text command: %v\n", err)

			return
		}

		msg.BaseMessage = BaseMessage{Time: time.Now(), ChannelID: channel}
		msg.RepliedTo = append(msg.RepliedTo, event.EventID)

		_, err = plugin.SendMessage(msg, &SendOptions{CommandUser: event.Author.ID, CommandResponse: sensitive})
		if err != nil {
			log.Printf("lightning: failed to respond to text command: %v\n", err)
		}
	}

	handleCommandEvent(bot, &CommandEvent{
		CommandOptions: &CommandOptions{
			event.BaseMessage, make(map[string]string), event.Author, bot, reply, bot.prefix,
		},
		Command: args[0],
		Options: args[1:],
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
			command = cmd
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
