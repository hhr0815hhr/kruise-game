package options

type GssOptions struct {
	Enable          bool               `toml:"enable"`
	HostPortOptions GssHostPortOptions `toml:"hostPort"`
}

type GssHostPortMap struct {
	Inner string `toml:"inner"`
	Outer string `toml:"outer"`
}

type GssHostPortRange struct {
	MaxPort int32 `toml:"max_port"`
	MinPort int32 `toml:"min_port"`
}

type GssHostPortOptions struct {
	Range GssHostPortRange `toml:"range"`
	IpMap []GssHostPortMap `toml:"ipMap"`
}

func (v GssOptions) Valid() bool {
	hostportOptions := v.HostPortOptions
	if hostportOptions.Range.MaxPort <= hostportOptions.Range.MinPort {
		return false
	}
	if hostportOptions.Range.MinPort <= 0 {
		return false
	}
	if len(hostportOptions.IpMap) == 0 {
		return false
	}
	return true
}

func (v GssOptions) Enabled() bool {
	return v.Enable
}
