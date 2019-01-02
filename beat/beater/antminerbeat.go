package beater

import (
	"fmt"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/nomadit/antminerbeat/beat/bkloud"
	"github.com/nomadit/antminerbeat/beat/config"
	"github.com/nomadit/antminerbeat/beat/db"
	"github.com/nomadit/antminerbeat/beat/scan"
)

type Antminerbeat struct {
	done chan interface{}

	minerStatusScan *scan.MinerStatusScan
	minerNameScan   *scan.MinerNameScan
	syncMiner       *db.SyncMiner
	executor        *bkloud.Executor
	logger          *logp.Logger
}

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &Antminerbeat{
		done:   make(chan interface{}),
		logger: logp.NewLogger(b.Info.Beat),
	}

	server := db.NewServer(&c.Server)
	bt.minerStatusScan = scan.NewMinerStatusScan(&c.MinerScan, b.Publisher, server)
	bt.minerNameScan = scan.NewMinerNameScan(&c.MinerScan, server)
	bt.syncMiner = db.NewSyncMiner(&c.Server, server)
	bt.syncMiner.AddScan(bt.minerStatusScan)
	bt.syncMiner.AddScan(bt.minerNameScan)

	var err error
	bt.executor, err = bkloud.NewExecutor(&c.Server, &c.MinerScan, server)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

func (bt *Antminerbeat) Run(b *beat.Beat) error {
	bt.logger.Info("antminerbeat is running! Hit CTRL-C to stop it.")
	bt.logger.Info("Beat RUN")

	if err := bt.minerStatusScan.Start(); err != nil {
		return err
	}
	defer bt.minerStatusScan.Stop()

	if err := bt.minerNameScan.Start(); err != nil {
		return err
	}
	defer bt.minerNameScan.Stop()

	if err := bt.syncMiner.Start(); err != nil {
		return err
	}
	defer bt.syncMiner.Stop()

	if err := bt.executor.Start(); err != nil {
		return err
	}
	defer bt.executor.Stop()

	<-bt.done
	bt.logger.Info("Shutting down.")
	return nil
}

func (bt *Antminerbeat) Stop() {
	close(bt.done)
}
