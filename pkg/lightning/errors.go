package lightning

// ChannelDisabler is an interface that allows a channel to be disabled in an external system.
type ChannelDisabler interface {
	Disable() *ChannelDisabled
}

// PluginRegisteredError only occurs when a plugin/type is already registered and
// can't be registered again.
type PluginRegisteredError struct{}

func (PluginRegisteredError) Error() string {
	return "plugin (or type) already registered: this is a bug or misconfiguration"
}

// MissingPluginError only occurs when a plugin/type is not found.
type MissingPluginError struct{}

func (MissingPluginError) Error() string {
	return "plugin not found internally: this is a bug or misconfiguration"
}

// PluginConfigError only occurs when a plugin is passed an invalid config on registration.
type PluginConfigError struct {
	Plugin  string
	Message string
}

func (p PluginConfigError) Error() string {
	return "plugin configuration error: " + p.Plugin + ": " + p.Message
}

// PluginMethodError is a wrapped error that occurs when a plugin method fails.
type PluginMethodError struct {
	err     error
	ID      string
	Method  string
	Message string
}

func (p PluginMethodError) Error() string {
	return "plugin " + p.ID + " method " + p.Method + " failed: " + p.Message + ": " + p.err.Error()
}

func (p PluginMethodError) Unwrap() error {
	return p.err
}
