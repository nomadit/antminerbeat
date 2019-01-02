// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Server      ServerInfo        `config:"server"`
	NetworkScan NetworkScanConfig `config:"scan.network"`
	MinerScan   MinerScanConfig   `config:"scan.miner.status"`
}

type ServerInfo struct {
	ServerHost string        `config:"server_host"`
	Period     time.Duration `config:"period"`
}

type NetworkScanConfig struct {
	Workers int           `config:"workers"`
	Jobs    int           `config:"jobs"`
	Results int           `config:"results"`
	Period  time.Duration `config:"period"`
}

type MinerScanConfig struct {
	Workers         int           `config:"workers"`
	Period          time.Duration `config:"period"`
	DefaultUser     string        `config:"default_user"`
	DefaultPassword string        `config:"default_password"`
}

var DefaultConfig = Config{
	Server: ServerInfo{
		ServerHost: "localhost:3001",
		Period:     10 * time.Second,
	},
	NetworkScan: NetworkScanConfig{
		Workers: 10,
		Jobs:    200,
		Results: 200,
		Period:  10 * time.Second,
	},
	MinerScan: MinerScanConfig{
		Workers:         10,
		Period:          10 * time.Second,
		DefaultUser:     "root",
		DefaultPassword: "root",
	},
}
