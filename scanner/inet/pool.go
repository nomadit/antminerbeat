package inet

import (
	"bytes"
	"github.com/nomadit/antminerbeat/scanner/config"
	"github.com/nomadit/antminerbeat/scanner/db"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

func newNmapPool(conf *config.NetworkScanConfig, remoteServer *db.Server, proxy *db.Proxy) *pool {
	return &pool{
		workers:      conf.Workers,
		jobs:         make(chan string, conf.Jobs),
		results:      make(chan string, conf.Results),
		remoteServer: remoteServer,
		proxy: proxy,
	}
}

type pool struct {
	workers      int
	jobs         chan string
	results      chan string
	wg           sync.WaitGroup
	remoteServer *db.Server
	proxy        *db.Proxy
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
			cmd := exec.Command("nmap", "-T4", "-sn", "--max-retries", strconv.Itoa(1), job)
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Foreground: false,
				Setsid:     true,
			}
			var output bytes.Buffer
			cmd.Stdout = &output

			// Start command asynchronously
			if err := cmd.Start(); err != nil {
				log.Println("cmd.Start", err)
				return
			}

			if err := cmd.Wait(); err != nil {
				log.Println("cmd.Wail", err)
				return
			}
			cmd.Process.Kill()
			p.results <- output.String()
		}()
	}
}

func (p *pool) collection() {
	for result := range p.results {
		if strings.Contains(result, "(0 hosts up)") == false {
			macIPMap := p.filterNmapWithIpTable(result)
			for key, value := range *macIPMap {
				var m db.Miner
				if miner, ok := db.IpTable.Find(key); !ok {
					log.Println("new value", key, value)
					m = db.Miner{
						IP:      value,
						Mac:     key,
						IsValid: true,
					}
					db.IpTable.Set(key, m)
					p.sendUpsert(m)
				} else {
					m = miner.(db.Miner)
					if m.IP != value {
						log.Println("upsert value", key, value)
						m.IP = value
						db.IpTable.Set(key, m)
						p.sendUpsert(m)
					}
				}
			}
		}
	}
}

func (p *pool) sendUpsert(item interface{}) {
	p.remoteServer.UpsertPc(p.proxy.ID, item)
}

func (p *pool) filterNmapWithIpTable(str string) *map[string]string {
	ipReg, err := regexp.Compile("Nmap scan report for ([a-zA-Z0-9.]+)")
	if err != nil {
		return nil
	}
	macReg, err := regexp.Compile("MAC Address: (([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})).*")
	if err != nil {
		return nil
	}
	inlist := strings.Split(str, "\n")
	macIPMap := map[string]string{}
	var ip string
	for _, line := range inlist {
		res := ipReg.FindStringSubmatch(line)
		if res != nil {
			ip = res[1]
		}
		res = macReg.FindStringSubmatch(line)
		if res != nil {
			macIPMap[res[1]] = ip
		}
	}
	for _, m := range *db.IpTable.GetValidList() {
		macIPMap[m.Mac] = m.IP
	}
	return &macIPMap
}
