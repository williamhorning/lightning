package lightning

// ChannelDisabler is an interface that allows a channel to be disabled in an external system.
type ChannelDisabler interface {
	Disable() *ChannelDisabled
}

// PluginRegisteredError only occurs when a plugin is already registered and can't be registered again.
type PluginRegisteredError struct{}

func (PluginRegisteredError) Error() string {
	return "plugin already registered: this is a bug or misconfiguration"
}

// MissingPluginError only occurs when a plugin/type is not found.
type MissingPluginError struct{}

func (MissingPluginError) Error() string {
	return "plugin not found internally: this is a bug or misconfiguration"
}

// PluginMethodError is a wrapped error that occurs when a plugin method fails.
type PluginMethodError struct {
	ID      string
	Method  string
	Message string
	err     []error
}

func (p PluginMethodError) Error() string {
	str := "plugin " + p.ID + " method " + p.Method + " failed: " + p.Message + ": "

	for _, err := range p.err {
		str += "\n\t" + err.Error()
	}

	return str
}

func (p PluginMethodError) Unwrap() []error {
	return p.err
}
