package inet

import (
	"errors"
	"fmt"
	"github.com/nomadit/antminerbeat/scanner/config"
	"github.com/nomadit/antminerbeat/scanner/db"
	"github.com/nomadit/antminerbeat/scanner/ifconfig"
	"log"
	"strconv"
	"strings"
	"time"
)

func NewNetworkScan(conf *config.Config) (*NetworkScan, error) {
	networks := ifconfig.PrivateNetworks()
	if len(*networks) != 1 {
		return nil, fmt.Errorf("the length of private networks is not one(real len: %d)", len(*networks))
	}

	network := (*networks)[0]
	remoteServer := db.NewServer(&conf.Server)
	if remoteServer == nil {
		log.Fatal("remoteServer is not initialized")
		return nil, errors.New("remoteServer is not initialized")
	}
	serial := readSerialKey()
	proxy, err := remoteServer.InitProxy(&network, serial)
	if err != nil {
		log.Fatal("Proxy is not initialized", err)
		return nil, errors.New("proxy is not initialized")
	}
	go checkNUpdateSerialKey(serial, proxy.SerialKey)
	scan := NetworkScan{
		network: &network.Network,
		period:  conf.NetworkScan.Period,
		running: false,
		done:    make(chan interface{}),
	}
	scan.scanPool = newNmapPool(&conf.NetworkScan, remoteServer, proxy)
	go scan.scanPool.workerPool()
	go scan.scanPool.collection()
	return &scan, nil
}

type NetworkScan struct {
	network  *string
	period   time.Duration
	running  bool
	done     chan interface{}
	scanPool *pool
}

func (s *NetworkScan) Start() error {
	log.Println("Start network scan")
	if s.running {
		return errors.New("network scanner already running")
	}
	s.running = true
	go s.run()
	return nil
}

func (s *NetworkScan) Stop() error {
	if !s.running {
		return errors.New("network scanner already stopped")
	}
	s.running = false
	close(s.done)
	return nil
}

//func (s *NetworkScan) Add(scan scan.Scan) {
//	s.scanPool.add(scan)
//}

func (s *NetworkScan) run() {
	subNetworks, err := splitSubNetworks(*s.network)
	if err != nil {
		log.Fatal("splitSubNetworks", err)
		return
	}

	var timer *time.Timer
	timer = time.NewTimer(0)
	for {
		select {
		case <-timer.C:
			log.Println("RUN SCAN NETWORK")
			jobs := make([]string, 0)
			for _, network := range *subNetworks {
				jobs = append(jobs, network)
			}

			for _, job := range jobs {
				s.scanPool.wg.Add(1)
				s.scanPool.jobs <- job
			}
			s.scanPool.wg.Wait()
		case <-s.done:
			log.Println("network scan done")
			return
		}
		if timer != nil {
			timer.Stop()
		}
		timer = time.NewTimer(s.period)
	}
}

func splitSubNetworks(network string) (*[]string, error) {
	items := strings.Split(network, "/")
	if len(items) != 2 {
		return nil, fmt.Errorf("items size is wrong %s when split with /", network)
	}
	networks := []string{}
	switch items[1] {
	case "24":
		networks = append(networks, network)
		return &networks, nil
	case "16":
		ipEntities := strings.Split(items[0], ".")
		base := ipEntities[0] + "." + ipEntities[1] + "."
		for i := 0; i < 256; i++ {
			networks = append(networks, base+strconv.Itoa(i)+".1/24")
		}
		return &networks, nil
	}
	return nil, fmt.Errorf("not b class or c class %s", network)
}
