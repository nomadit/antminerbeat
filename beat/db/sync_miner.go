package db

import (
	"errors"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/nomadit/antminerbeat/beat/config"
	"time"
)

var resetCountOfErrorNoState = 10

func NewSyncMiner(info *config.ServerInfo, remoteServer *Server) *SyncMiner {
	return &SyncMiner{
		remoteServer: remoteServer,
		period:       info.Period,
		running:      false,
		done:         make(chan interface{}),
		logger:       logp.NewLogger("syncDB"),
	}
}

type SyncMiner struct {
	remoteServer *Server
	scanners     []Scan
	period       time.Duration
	done         chan interface{}
	running      bool
	logger       *logp.Logger
}

func (s *SyncMiner) Start() error {
	if s.running {
		return errors.New("network scanner already running")
	}
	s.running = true
	go s.run()
	return nil
}

func (s *SyncMiner) AddScan(scan Scan) {
	s.scanners = append(s.scanners, scan)
}


func (s *SyncMiner) run() {
	var timer *time.Timer
	timer = time.NewTimer(0)
	count := 0
	for {
		select {
		case <-timer.C:
			// if sync.Map has the length, consider next.
			// after comparing a length of list and sync.Map if they are different, then remove the sync.Map
			IpTable.clear()
			list := s.remoteServer.getAllMacs()
			if list != nil && len(*list) > 0 {
				miners := s.makeMiners(list, count)
				IpTable.SetTable(miners)
				for _, scan := range s.scanners {
					for _, miner := range *miners {
						scan.Add(&miner)
					}
				}
			}
			count++
			// check the really state about the error_no_worker state for each 10 times
			if count > resetCountOfErrorNoState {
				count = 0
			}
		case <-s.done:
			s.logger.Info("db sync done")
			return
		}
		if timer != nil {
			timer.Stop()
		}
		timer = time.NewTimer(s.period)
	}
}

func (s *SyncMiner) Stop() error {
	close(s.done)
	return nil
}

func (s *SyncMiner) makeMiners(list *[]pc, count int) *[]Miner {
	table := make([]Miner, 0)
	for _, item := range *list {
		isValid := true
		if item.Status != "RUN" && count < resetCountOfErrorNoState {
			isValid = false
		}
		if item.Status == "STOP" || item.DeletedAt.Valid {
			isValid = false
		}
		row := Miner{
			ID:       item.ID,
			IP:       item.IP,
			Mac:      item.MacAddress,
			Status:   item.Status,
			Name:     item.Name,
			User:     item.User,
			Password: item.Password,
			IsValid:  isValid,
		}
		table = append(table, row)
	}
	return &table
}

