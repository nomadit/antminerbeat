// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Server      ServerInfo        `mapstructure:"server"`
	NetworkScan NetworkScanConfig `mapstructure:"scan"`
}

type ServerInfo struct {
	ServerHost string `mapstructure:"server_host"`
	Period     time.Duration
}

type NetworkScanConfig struct {
	Workers int
	Jobs    int
	Results int
	Period  time.Duration
}

