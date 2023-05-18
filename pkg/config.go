package pkg

// Config is user plugin configuration
type Config struct {
}

// DefaultConfig implements plugin.Configurer
func (m *Module) DefaultConfig() interface{} {
	return &Config{}
}

func (m *Module) GetConfig() interface{} {
	return m.config
}

// ValidateAndSetConfig implements plugin.Configurer
func (m *Module) ValidateAndSetConfig(config interface{}) error {
	newConfig := config.(*Config)
	m.config = newConfig
	return nil
}
