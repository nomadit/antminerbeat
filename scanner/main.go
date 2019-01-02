package main

import (
	"flag"
	"github.com/nomadit/antminerbeat/scanner/config"
	"github.com/nomadit/antminerbeat/scanner/inet"
	"github.com/spf13/viper"
	"log"
)

var (
	ServiceMode = flag.String("mode", "dev", "The service mode")
	ConfigFilePath = flag.String("config", "scanner.yaml", "the service config")
)

func main() {
	flag.Parse()
	confMap := getConf()
	conf := (*confMap)[*ServiceMode]
	netScan, err := inet.NewNetworkScan(&conf)
	if err != nil {
		log.Println("scannet", err)
		return
	}
	if err := netScan.Start(); err != nil {
		log.Println("scannet", err)
		return
	}
	done := make(chan string)
	<-done
}

func getConf() *map[string]config.Config {
	viper.SetConfigFile(*ConfigFilePath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	conf := map[string]config.Config{}
	err := viper.Unmarshal(&conf)
	if err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}
	return &conf
}

