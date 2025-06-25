package lightning

import (
	"fmt"
	"strings"
	"time"
)

var (
	commandRegistry = make(map[string]Command)
	CommandPrefix   = ""
)

func RegisterCommand(command Command) {
	Log.Debug().Str("command", command.Name).Msg("Registering command")
	commandRegistry[command.Name] = command
	commands := make([]Command, 0, len(commandRegistry))
	for _, cmd := range commandRegistry {
		commands = append(commands, cmd)
	}

	for _, plugin := range Plugins.Plugins {
		if err := plugin.SetupCommands(commands); err != nil {
			LogError(err, "Failed to setup commands for plugin", map[string]any{
				"plugin": plugin.Name(),
			}, ReadWriteDisabled{})
		}
	}
}

func GetCommand(name string) (Command, bool) {
	command, exists := commandRegistry[name]
	return command, exists
}

type CommandArgument struct {
	Name        string
	Description string
	Required    bool
}

type CommandOptions struct {
	BaseMessage
	Arguments map[string]string
	Prefix    string
}

type Command struct {
	Name        string
	Description string
	Arguments   []CommandArgument
	Subcommands []Command
	Executor    func(options CommandOptions) (string, error)
}

type CommandEvent struct {
	CommandOptions
	Command    string
	Subcommand *string
	Options    *[]string
	Reply      func(message string) error
}

func HelpCommand() Command {
	return Command{
		Name:        "help",
		Description: "get help with the bot",
		Arguments:   []CommandArgument{},
		Subcommands: []Command{},
		Executor: func(options CommandOptions) (string, error) {
			return "hi! i'm lightning v0.8.0-alpha.9.\ncheck out [the docs](https://williamhorning.eu.org/lightning/) for help!", nil
		},
	}
}

func PingCommand() Command {
	return Command{
		Name:        "ping",
		Description: "check if the bot is alive",
		Arguments:   []CommandArgument{},
		Subcommands: []Command{},
		Executor: func(options CommandOptions) (string, error) {
			return fmt.Sprintf("Pong! 🏓 %dms", (time.Since(options.Time)).Milliseconds()), nil
		},
	}
}

func SetupCommands(prefix string) {
	CommandPrefix = prefix

	RegisterCommand(HelpCommand())
	RegisterCommand(PingCommand())

	go func() {
		for event := range Plugins.ListenCommands() {
			handleCommandEvent(event)
		}
	}()

	go func() {
		for event := range Plugins.ListenMessages() {
			handleMessageCommand(event, prefix)
		}
	}()
}

func handleMessageCommand(event Message, prefix string) {
	if !strings.HasPrefix(event.Content, prefix) {
		return
	}

	Log.Trace().Str("event_id", event.EventID).Str("plugin", event.Plugin).Msg("Handling command message")

	content := strings.TrimPrefix(event.Content, prefix)
	args := strings.Fields(content)
	if len(args) == 0 {
		return
	}

	commandName := args[0]
	options := args[1:]

	handleCommandEvent(CommandEvent{
		CommandOptions: CommandOptions{
			Arguments:   make(map[string]string),
			BaseMessage: event.BaseMessage,
			Prefix:      prefix,
		},
		Command: commandName,
		Options: &options,
		Reply: func(message string) error {
			plugin, exists := Plugins.Get(event.Plugin)
			if !exists {
				return LogError(ErrPluginNotFound, "Plugin not found for command reply", map[string]any{
					"plugin": event.Plugin,
					"event":  event.EventID,
				}, ReadWriteDisabled{})
			}

			msg := CreateMessage(message)
			msg.ChannelID = event.ChannelID
			_, err := plugin.SendMessage(msg, nil)
			return err
		},
	})
}

func handleCommandEvent(event CommandEvent) error {
	Log.Trace().Str("event_id", event.EventID).Str("command", event.Command).Msg("Handling command event")

	command, exists := GetCommand(event.Command)
	if !exists {
		Log.Trace().Str("command", event.Command).Msg("Command not found, using help command")
		command = HelpCommand()
	}

	if event.Options != nil && len(*event.Options) > 0 {
		event.Subcommand = &(*event.Options)[0]
		*event.Options = (*event.Options)[1:]
	}

	for _, subcommand := range command.Subcommands {
		if event.Subcommand != nil && subcommand.Name == *event.Subcommand {
			command = subcommand
			break
		}
	}

	for _, arg := range command.Arguments {
		if event.CommandOptions.Arguments[arg.Name] == "" && event.Options != nil && len(*event.Options) > 0 {
			event.CommandOptions.Arguments[arg.Name] = (*event.Options)[0]
			*event.Options = (*event.Options)[1:]
		}

		if arg.Required && event.CommandOptions.Arguments[arg.Name] == "" {
			Log.Trace().Str("argument", arg.Name).Msg("Required argument missing")
			err := event.Reply("Please provide the " + arg.Name + " argument. Try using the `" + event.Prefix + "help` command.")
			if err != nil {
				return LogError(err, "Error sending missing argument response", map[string]any{
					"argument": arg.Name,
					"command":  command.Name,
					"event":    event.EventID,
				}, ReadWriteDisabled{})
			}
			return nil
		}
	}

	response, err := command.Executor(event.CommandOptions)

	if err != nil {
		response = LogError(err, "Error executing command", map[string]any{
			"command": command.Name,
			"event":   event.EventID,
		}, ReadWriteDisabled{}).Error()
	}

	if err = event.Reply(response); err != nil {
		return LogError(err, "Error sending command response", map[string]any{
			"command": command.Name,
			"event":   event.EventID,
		}, ReadWriteDisabled{})
	}

	Log.Trace().Str("event_id", event.EventID).Str("command", command.Name).Msg("Command handled successfully")

	return nil
}
