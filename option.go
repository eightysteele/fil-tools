package filecoin

// StoreOption mutates a Filecoin configuration
type FilecoinOption func(*FilecoinConfig) error

// FilecoinConfig has configuration parameters for a store
type FilecoinConfig struct {
	Debug bool
}

func defaultConfig() *FilecoinConfig {
	return &FilecoinConfig{}
}

func WithDebug(enabled bool) FilecoinOption {
	return func(c *FilecoinConfig) error {
		c.Debug = enabled
		return nil
	}
}
