package bkloud

import (
	"errors"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/nomadit/antminerbeat/beat/config"
	"github.com/nomadit/antminerbeat/beat/db"
	"time"
)

func NewExecutor(conf *config.ServerInfo, antConf *config.MinerScanConfig, remoteServer *db.Server) (*Executor, error) {
	logger := logp.NewLogger("Exe cmd")
	if remoteServer == nil {
		logger.Fatal("remoteServer is not initialized")
		return nil, errors.New("remoteServer is not initialized")
	}
	exec := Executor{
		period:          conf.Period,
		running:         false,
		done:            make(chan interface{}),
		logger:          logger,
		remoteServer:    remoteServer,
		defaultUser:     antConf.DefaultUser,
		defaultPassword: antConf.DefaultPassword,
	}
	exec.pool = newNmapPool(100, 200, 200, remoteServer)
	go exec.pool.workerPool()
	go exec.pool.collection()
	return &exec, nil
}

type Executor struct {
	period          time.Duration
	running         bool
	pool            *pool
	done            chan interface{}
	logger          *logp.Logger
	defaultUser     string
	defaultPassword string
	remoteServer    *db.Server
}

func (s *Executor) Start() error {
	s.logger.Info("Start network scan")
	if s.running {
		return errors.New("network scanner already running")
	}
	s.running = true
	go s.run()
	return nil
}

func (s *Executor) Stop() error {
	if !s.running {
	}
	s.running = false
	close(s.done)
	return nil
}

func (s *Executor) run() {
	var timer *time.Timer
	timer = time.NewTimer(0)
	for {
		select {
		case <-timer.C:
			list := db.IpTable.GetValidList()
			if len(*list) > 0 {
				var ids []int64
				idMap := map[int64]db.Miner{}
				for _, item := range *list {
					ids = append(ids, item.ID)
					idMap[item.ID] = item
				}
				jobs := s.remoteServer.GetCommands(&ids)

				if jobs != nil {
					for _, job := range *jobs {
						s.pool.wg.Add(1)
						job.IP = idMap[job.PcID].IP
						s.pool.jobs <- Command{job, &s.defaultUser, &s.defaultPassword}
					}
					s.pool.wg.Wait()
				}
			}
		case <-s.done:
			s.logger.Info("network scan done")
			return
		}
		if timer != nil {
			timer.Stop()
		}
		timer = time.NewTimer(s.period)
	}
}
