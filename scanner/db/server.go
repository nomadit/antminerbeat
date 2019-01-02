package db

import (
	"bytes"
	"encoding/json"
	"github.com/nomadit/antminerbeat/scanner/config"
	"github.com/nomadit/antminerbeat/scanner/ifconfig"
	"github.com/parnurzeal/gorequest"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const httpPrefix = "http://"

func NewServer(info *config.ServerInfo) *Server {
	return &Server{
		serverHost: info.ServerHost,
	}
}

type Server struct {
	serverHost string
}

func (r *Server) InitProxy(network *ifconfig.Ifaddr, serialKey string) (*Proxy, error) {
	net := r.getNetwork(network.Mac)
	if net == nil {
		var err error
		proxy := &Proxy{
			ID:         0,
			Network:    network.Network,
			MacAddress: network.Mac,
			SerialKey:  serialKey,
		}

		if net, err = r.insertNetwork(proxy); err != nil {
			return nil, err
		}
	} else {
		if strings.Compare(network.Network, net.Network) != 0 {
			net.Network = network.Network
			go r.updateNetwork(net)
		}
	}
	return net, nil
}

func (r *Server) getNetwork(mac string) *Proxy {
	var ret Proxy
	resp, body, errs := gorequest.New().Get(httpPrefix + r.serverHost + "/api/beat/proxy/by_mac/" + mac).EndStruct(&ret)
	if errs != nil {
		log.Println(errs)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		log.Println(body)
		return nil
	}
	return &ret
}

func (r *Server) updateNetwork(proxy *Proxy) bool {
	resp, body, errs := gorequest.New().Put(httpPrefix + r.serverHost + "/api/beat/proxy").
		Send(proxy).
		End()
	if errs != nil {
		log.Println(errs)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		log.Println(body)
		return false
	}
	return true
}

func (r *Server) insertNetwork(proxy *Proxy) (*Proxy, error) {
	resp, body, errs := gorequest.New().Post(httpPrefix + r.serverHost + "/api/beat/proxy").
		Send(proxy).
		End()
	if errs != nil {
		var buf bytes.Buffer
		for _, err := range errs {
			buf.WriteString(err.Error())
		}
		return nil, errors.New(buf.String())
	}
	if resp.StatusCode != http.StatusOK {
		log.Println(body)
		return nil, errors.New(resp.Status + ":serial_key:" + proxy.SerialKey)
	}
	err := json.Unmarshal([]byte(body), proxy)
	if err != nil {
		return nil, err
	}
	return proxy, nil
}

func (r *Server) UpsertPc(id int64, item interface{}) bool {
	resp, body, errs := gorequest.New().
		Put(httpPrefix + r.serverHost + "/api/beat/pc/by_net_id/" + strconv.FormatInt(id, 10)).
		Send(item).
		End()
	if errs != nil {
		log.Println(errs)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		log.Println(body)
		return false
	}
	return true
}

type Proxy struct {
	ID         int64  `json:"id"`
	MacAddress string `json:"macAddress"`
	Network    string `json:"network"`
	SerialKey  string `json:"serialKey"`
}

type Miner struct {
	ID       int64   `json:"id"`
	IP       string  `json:"ip"`
	Mac      string  `json:"macAddress"`
	Status   string  `json:"status"`
	User     *string `json:"user"`
	Password *string `json:"password"`
	IsValid  bool
	//request *gorequest.SuperAgent
}
