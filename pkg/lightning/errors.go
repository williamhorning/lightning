package lightning

// PluginRegisteredError only occurs when a plugin is already registered and can't be registered again.
type PluginRegisteredError struct {
	Name string
}

func (p PluginRegisteredError) Error() string {
	return "a plugin with the name " + p.Name + " has already been registered"
}

// MissingPluginTypeError only occurs when a plugin type is not found.
type MissingPluginTypeError struct {
	Name string
}

func (p MissingPluginTypeError) Error() string {
	return "can't make an instance of plugin type " + p.Name + " because it has not been registered"
}

// MissingPluginInstanceError only occurs when a plugin is not found.
type MissingPluginInstanceError struct {
	Name string
}

func (p MissingPluginInstanceError) Error() string {
	return "can't call a method for plugin " + p.Name + " because it does not exist"
}

// PluginMethodError is a wrapped error that occurs when a plugin method fails.
type PluginMethodError struct {
	ID     string
	Method string
	err    error
}

func (p PluginMethodError) Error() string {
	return p.Method + " failed in " + p.ID + ": " + p.err.Error()
}

func (p PluginMethodError) Unwrap() error {
	return p.err
}
