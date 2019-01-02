package scan

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/nomadit/antminerbeat/beat/config"
	"github.com/nomadit/antminerbeat/beat/db"
	"github.com/nomadit/antminerbeat/beat/digestRequest"
	"strings"
	"sync"
	"time"
)

func NewMinerNameScan(conf *config.MinerScanConfig, remoteServer *db.Server) *MinerNameScan {
	return &MinerNameScan{
		period:          conf.Period,
		running:         false,
		done:            make(chan interface{}),
		add:             make(chan *job),
		rm:              make(chan string),
		remoteServer:    remoteServer,
		defaultUser:     conf.DefaultUser,
		defaultPassword: conf.DefaultPassword,
		logger:          logp.NewLogger("minerStatusScan"),
	}
}

type MinerNameScan struct {
	period          time.Duration
	running         bool
	done            chan interface{}
	add             chan *job
	rm              chan string
	macJobMap       sync.Map
	remoteServer    *db.Server
	defaultUser     string
	defaultPassword string
	logger          *logp.Logger
}

func (l *MinerNameScan) Start() error {
	if l.running {
		return errors.New("scheduler already running")
	}
	l.running = true
	go l.run()
	return nil
}

func (l *MinerNameScan) Add(m *db.Miner) {
	var j *job
	if _, ok := l.macJobMap.Load(m.Mac); !ok && m.IsValid {
		j = &job{
			id:       m.ID,
			mac:      m.Mac,
			ip:       m.IP,
			status:   m.Status,
			isValid:  m.IsValid,
			hostname: m.Name,
			user:     m.User,
			password: m.Password,
			finish:   true,
			first:    true,
		}
		if m.User == nil {
			j.user = &l.defaultUser
			j.password = &l.defaultPassword
		}
		if !l.running {
			l.doAdd(j)
		} else {
			l.add <- j
		}
	}
}

func (l *MinerNameScan) Remove(mac string) {
	if val, ok := l.macJobMap.Load(mac); ok {
		if val.(job).isValid {
			if !l.running {
				l.doRemove(mac)
			} else {
				l.rm <- mac
			}
		}
	}
}

func (l *MinerNameScan) run() {
	var timer *time.Timer
	timer = time.NewTimer(0)
	var count int64 = 0
	for {
		select {
		case <-timer.C:
			count++
			l.macJobMap.Range(func(key, value interface{}) bool {
				j := value.(job)
				if j.isValid && j.finish {
					j.finish = false
					l.macJobMap.Store(j.mac, j)
					go l.scan(j)
				}
				return true
			})
		case <-l.done:
			l.logger.Info("log scan done")
			return
		case j := <-l.add:
			l.doAdd(j)
		case mac := <-l.rm:
			l.doRemove(mac)
		}
		if timer != nil {
			timer.Stop()
		}
		duration := l.period * time.Duration(count)
		timer = time.NewTimer(duration)
		if count > 1000000000 {
			count = 0
		}
	}
}

func (l *MinerNameScan) scan(j job) {
	l.logger.Info("Name San\t", j.mac)
	uri := "/cgi-bin/get_network_info.cgi"
	hostTemplate := "http://%s"
	host := fmt.Sprintf(hostTemplate, j.ip)
	resp, err := digestRequest.Digest("GET", host, uri, *j.user, *j.password, nil)

	if err != nil || resp == nil {
		l.Remove(j.mac)
		return
	} else {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		result := map[string]string{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			l.logger.Info(err)
			l.Remove(j.mac)
		} else {
			hostname := result["conf_hostname"]
			if j.hostname == nil || *j.hostname != hostname {
				j.hostname = &hostname
				l.remoteServer.UpdateName(j.id, j.hostname)
				j.finish = true
				l.macJobMap.Store(j.mac, j)
			}
		}
	}
	resp.Body.Close()
}

func (l *MinerNameScan) doAdd(j *job) {
	if existedJob, ok := l.macJobMap.Load(j.mac); !ok {
		l.macJobMap.Store(j.mac, *j)
	} else if strings.Compare(existedJob.(job).ip, j.ip) != 0 {
		l.macJobMap.Store(j.mac, *j)
	} else if existedJob.(job).finish && j.isValid {
		l.macJobMap.Store(j.mac, *j)
	}
}

func (l *MinerNameScan) doRemove(mac string) {
	if _, ok := l.macJobMap.Load(mac); ok {
		l.macJobMap.Delete(mac)
	}
}

func (l *MinerNameScan) Stop() error {
	if !l.running {
		return errors.New("miner status scan already stopped")
	}
	l.running = false
	close(l.done)
	return nil
}
