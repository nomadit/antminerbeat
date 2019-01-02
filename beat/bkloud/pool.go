package bkloud

import (
	"github.com/elastic/beats/libbeat/logp"
	"github.com/nomadit/antminerbeat/beat/db"
	"sync"
)

func newNmapPool(workers int, jobs int, result int, remoteServer *db.Server) *pool {
	return &pool{
		workers:      workers,
		jobs:         make(chan Command, jobs),
		results:      make(chan Command, result),
		remoteServer: remoteServer,
		logger:       logp.NewLogger("netpool"),
	}
}

type pool struct {
	workers      int
	jobs         chan Command
	results      chan Command
	wg           sync.WaitGroup
	scanners     []db.Scan
	remoteServer *db.Server
	logger       *logp.Logger
}

func (p *pool) workerPool() {
	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go p.worker(&wg)
	}
	wg.Wait()
}

func (p *pool) worker(wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range p.jobs {
		func() {
			defer p.wg.Done()
			err := job.run()
			if err != nil {
				p.logger.Error(err)
			}

			p.results <- job
		}()
	}
}

func (p *pool) collection() {
	for job := range p.results {
		p.remoteServer.UpdateStateOfCommand(job.ID, job.Status)
	}
}

