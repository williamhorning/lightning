package lightning

// BotError is the wrapper for any error encountered by a [Bot].
type BotError struct {
	Disable *ChannelDisabled

	underlying error
	message    string
}

func (botErr BotError) Error() string {
	return botErr.message
}

func (botErr BotError) Unwrap() error {
	return botErr.underlying
}

// PluginRegisteredError only occurs when a plugin/type is already registered and
// can't be registered again.
type PluginRegisteredError struct{}

func (PluginRegisteredError) Error() string {
	return "plugin (or type) already registered: this is a bug or misconfiguration"
}

// MissingPluginError only occurs when a plugin/type is not found and the action
// requires it to be found.
type MissingPluginError struct{}

func (MissingPluginError) Error() string {
	return "plugin not found internally: this is a bug or misconfiguration"
}

// PluginConfigError only occurs when a plugin is passed an invalid config on registration.
type PluginConfigError struct{}

func (PluginConfigError) Error() string {
	return "plugin config is invalid"
}

type nilLogError struct{}

func (nilLogError) Error() string {
	return "LogError called with nil error. Please provide a valid error"
}
