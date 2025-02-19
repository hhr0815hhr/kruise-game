package options

type GssOptions struct {
	Enable   bool               `toml:"enable"`
	HostPort GssHostPortOptions `toml:"hostPort"`
	IpMap    []GssHostPortMAp   `toml:"ipMap"`
}

type GssHostPortMAp struct {
	Inner string `toml:"inner"`
	Outer string `toml:"outer"`
}

type GssHostPortOptions struct {
	MaxPort int32 `toml:"max_port"`
	MinPort int32 `toml:"min_port"`
}

func (v GssOptions) Valid() bool {
	slbOptions := v.HostPort
	if slbOptions.MaxPort <= slbOptions.MinPort {
		return false
	}
	if slbOptions.MinPort <= 0 {
		return false
	}
	return true
}

func (v GssOptions) Enabled() bool {
	return v.Enable
}
