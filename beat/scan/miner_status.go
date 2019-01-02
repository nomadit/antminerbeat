package scan

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/nomadit/antminerbeat/beat/config"
	"github.com/nomadit/antminerbeat/beat/db"
	"github.com/nomadit/antminerbeat/beat/digestRequest"
	"golang.org/x/net/html"
	"net/url"
	"strings"
	"sync"
	"time"
)

func NewMinerStatusScan(conf *config.MinerScanConfig, pipeline beat.Pipeline, remoteServer *db.Server) *MinerStatusScan {
	return &MinerStatusScan{
		period:          conf.Period,
		pipeline:        pipeline,
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

type MinerStatusScan struct {
	pipeline        beat.Pipeline
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

func (l *MinerStatusScan) Start() error {
	if l.running {
		return errors.New("scheduler already running")
	}
	l.running = true
	go l.run()
	return nil
}

func (l *MinerStatusScan) Add(m *db.Miner) {
	var j *job
	if _, ok := l.macJobMap.Load(m.Mac); !ok && m.IsValid {
		j = &job{
			id:       m.ID,
			mac:      m.Mac,
			ip:       m.IP,
			status:   m.Status,
			isValid:  m.IsValid,
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

func (l *MinerStatusScan) Remove(mac string) {
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

func (l *MinerStatusScan) run() {
	var timer *time.Timer
	timer = time.NewTimer(0)
	client, err := l.pipeline.Connect()
	if err != nil {
		l.logger.Fatal("not connected")
	}
	for {
		select {
		case <-timer.C:
			l.macJobMap.Range(func(key, value interface{}) bool {
				j := value.(job)
				if j.isValid && j.finish {
					j.finish = false
					l.macJobMap.Store(j.mac, j)
					go l.scan(j, client)
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
		timer = time.NewTimer(l.period)
	}
}

func (l *MinerStatusScan) scan(j job, client beat.Client) {
	l.logger.Info("Status San\t", j.mac)
	uri := "/cgi-bin/minerStatus.cgi"
	hostTemplate := "http://%s"
	host := fmt.Sprintf(hostTemplate, j.ip)
	resp, err := digestRequest.Digest("GET", host, uri, *j.user, *j.password, nil)

	if err != nil || resp == nil {
		l.Remove(j.mac)
		return
	} else {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		if resultMap, err := l.findStatus(buf); err != nil {
			l.logger.Info(err)
			l.Remove(j.mac)
		} else {
			mapStr := common.MapStr{}
			mapStr.DeepUpdate(common.MapStr(*resultMap))
			event := beat.Event{
				Timestamp: time.Now(),
				Fields: common.MapStr{
					"beat.log": mapStr,
					"beat.mac": j.mac,
					"beat.id":  j.id,
				},
			}
			client.Publish(event)
			if j.status != "RUN" || j.status != "STOP" {
				j.status = "RUN"
				go l.remoteServer.UpdateState(j.id, j.status)
			}
			if j.first && mapStr["asic"] != nil && mapStr["conf"] != nil {
				asic := mapStr["asic"].([]map[string]interface{})
				if len(asic) != 0 {
					if l.remoteServer.UpdateConfig(j.id, asic[0]["freq"], mapStr["conf"]) {
						j.first = false
					}
				}
			}
			j.finish = true
			l.macJobMap.Store(j.mac, j)
		}
	}
	resp.Body.Close()
}

func (l *MinerStatusScan) findStatus(buf *bytes.Buffer) (*map[string]interface{}, error) {
	if len(buf.String()) == 0 {
		return nil, errors.New("empty")
	}
	content := buf.String()
	if strings.Contains(content, "Ant Miner") {
		return l.parseHtml(&content)
	} else {
		return nil, errors.New("is not ant miner")
	}
}

func (l *MinerStatusScan) doAdd(j *job) {
	if existedJob, ok := l.macJobMap.Load(j.mac); !ok {
		l.macJobMap.Store(j.mac, *j)
	} else if strings.Compare(existedJob.(job).ip, j.ip) != 0 {
		l.macJobMap.Store(j.mac, *j)
	} else if existedJob.(job).finish && j.isValid {
		l.macJobMap.Store(j.mac, *j)
	}
}

func (l *MinerStatusScan) doRemove(mac string) {
	err := db.IpTable.SetInValid(mac, false)
	if err != nil {
		l.logger.Error(err)
	}
	if _, ok := l.macJobMap.Load(mac); ok {
		go l.remoteServer.UpdateInValid(mac)
		l.macJobMap.Delete(mac)
	}
}

func (l *MinerStatusScan) Stop() error {
	if !l.running {
		return errors.New("miner status scan already stopped")
	}
	l.running = false
	close(l.done)
	return nil
}
func (l *MinerStatusScan) parseHtml(content *string) (*map[string]interface{}, error) {
	z := html.NewTokenizer(strings.NewReader(*content))
	result := map[string]interface{}{}
	isConfig := false
	configList := []map[string]string{}
	asicList := []map[string]interface{}{}
	for {
		tt := z.Next()

		if tt == html.ErrorToken {
			// End of the document, we're done
			result["conf"] = configList
			result["asic"] = asicList
			return &result, nil
		} else if tt == html.StartTagToken {
			t := z.Token()

			if t.Data == "div" {
				doc := z.Next()
				for _, attr := range t.Attr {
					if attr.Key == "id" && doc == html.TextToken {
						next := z.Token()
						val := next.Data
						switch attr.Val {
						case "ant_elapsed":
							result["elapsed"] = val
						case "ant_ghs5s":
							result["ghs5s"] = val
						case "ant_ghsav":
							result["ghsav"] = val
						case "ant_foundblocks":
							result["foundblocks"] = val
						case "ant_localwork":
							result["localwork"] = val
						case "ant_utility":
							result["utility"] = val
						case "ant_wu":
							result["wu"] = val
						case "ant_bestshare":
							result["bestshare"] = val
						case "cbi-table-1-url":
							u, err := url.Parse(val)
							if err != nil {
								l.logger.Error(err)
								isConfig = false
							} else if len(u.Hostname()) > 0 {
								isConfig = true
								item := map[string]string{"url": u.Hostname() + ":" + u.Port()}
								configList = append(configList, item)
							} else {
								isConfig = false
							}
						case "cbi-table-1-user":
							if isConfig {
								item := configList[len(configList)-1]
								if _, ok := item["user"]; !ok {
									item["user"] = val
								}
							}
						case "cbi-table-1-status":
							if isConfig {
								item := configList[len(configList)-1]
								if _, ok := item["status"]; !ok {
									item["status"] = val
								}
							}
						case "cbi-table-1-chain":
							item := map[string]interface{}{"idx": val}
							asicList = append(asicList, item)
						case "cbi-table-1-frequency":
							item := asicList[len(asicList)-1]
							if _, ok := item["freq"]; !ok {
								item["freq"] = val
							}
						case "cbi-table-1-rate":
							item := asicList[len(asicList)-1]
							if _, ok := item["rate"]; !ok {
								item["rate"] = val
							}
						case "cbi-table-1-hw":
							item := asicList[len(asicList)-1]
							if _, ok := item["hw"]; !ok {
								item["hw"] = val
							}
						case "cbi-table-1-temp":
							item := asicList[len(asicList)-1]
							if _, ok := item["temp_pcb"]; !ok {
								item["temp_pcb"] = val
							}
						case "cbi-table-1-temp2":
							item := asicList[len(asicList)-1]
							if _, ok := item["temp_chip"]; !ok {
								item["temp_chip"] = val
							}
						}
					}
				}
			}
			if t.Data == "td" {
				doc := z.Next()
				for _, attr := range t.Attr {
					if attr.Key == "id" && doc == html.TextToken {
						next := z.Token()
						val := next.Data
						switch attr.Val {
						case "ant_fan1":
							result["fan1"] = val
						case "ant_fan2":
							result["fan2"] = val
						}
					}
				}
			}
		}
	}
}
