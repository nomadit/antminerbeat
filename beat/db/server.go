package db

import (
	"fmt"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/go-sql-driver/mysql"
	"github.com/nomadit/antminerbeat/beat/config"
	"github.com/nomadit/antminerbeat/beat/ifconfig"
	"github.com/parnurzeal/gorequest"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
)

const httpPrefix = "http://"

func NewServer(info *config.ServerInfo) *Server {
	return &Server{
		serverHost: info.ServerHost,
		logger:     logp.NewLogger("remoteServer"),
	}
}

var proxy *Proxy

type Server struct {
	serverHost string
	//serialKey  string
	logger     *logp.Logger
}

func (r *Server) initProxy() error {
	networks := ifconfig.Networks()
	if len(*networks) != 1 {
		return fmt.Errorf("the length of networks is not one(real len: %d)", len(*networks))
	}
	network := (*networks)[0]

	proxy = r.getNetwork(&network.Mac)
	if proxy == nil {
		return errors.New("network is empty")
	}
	return nil
}

func (r *Server) getNetwork(mac *string) *Proxy {
	var ret Proxy
	resp, body, errs := gorequest.New().Get(httpPrefix + r.serverHost + "/api/beat/proxy/by_mac/" + *mac).EndStruct(&ret)
	if errs != nil {
		r.logger.Info(errs)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		r.logger.Info(http.StatusOK, string(body))
		return nil
	}
	return &ret
}

func (r *Server) getAllMacs() *[]pc {
	if proxy == nil {
		err := r.initProxy()
		if err != nil {
			r.logger.Info("Proxy info is empty\t", err.Error())
			return nil
		}
	}
	list := make([]pc, 0)
	idStr := strconv.FormatInt(proxy.ID, 10)
	resp, body, errs := gorequest.New().
		Get(httpPrefix + r.serverHost + "/api/beat/pc/all/by_net_id/" + idStr).EndStruct(&list)
	if errs != nil {
		r.logger.Info(errs)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		r.logger.Info(body)
		return nil
	}
	return &list
}

func (r *Server) GetCommands(ids *[]int64) *[]Command {
	list := make([]Command, 0)
	item := map[string]interface{}{}
	for _, id := range *ids {
		item["id[]"] = id
	}
	resp, body, errs := gorequest.New().Get(httpPrefix + r.serverHost + "/api/beat/command/list/by_pc_ids").
		Query(item).
		EndStruct(&list)
	if errs != nil {
		r.logger.Info(errs)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		r.logger.Info(body)
		return nil
	}
	return &list
}

func (r *Server) UpdateStateOfCommand(id int64, status string) error {
	item := map[string]interface{}{"status": status}
	resp, body, errs := gorequest.New().
		Put(httpPrefix + r.serverHost + "/api/beat/command/status/" + strconv.FormatInt(id, 10)).
		Send(item).
		End()
	if errs != nil {
		r.logger.Info(errs)
		return errs[0]
	}
	if resp.StatusCode != http.StatusOK {
		r.logger.Info(body)
		return errors.New(resp.Status)
	}
	return nil
}

func (r *Server) UpdateInValid(mac string) bool {
	resp, _, errs := gorequest.New().Put(httpPrefix + r.serverHost + "/api/beat/pc/invalid/by_mac/" + mac).End()
	if errs != nil {
		r.logger.Info(errs)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func (r *Server) UpdateState(id int64, status string) bool {
	item := map[string]string{"status": status}
	resp, _, errs := gorequest.New().Put(httpPrefix + r.serverHost + "/api/beat/pc/status/" + strconv.FormatInt(id, 10)).
		Send(item).
		End()
	if errs != nil {
		r.logger.Info(errs)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		r.logger.Info("response is error:", resp.StatusCode)
		return false
	}
	return true
}

func (r *Server) UpdateName(id int64, name *string) bool {
	item := map[string]*string{"name": name}
	resp, _, errs := gorequest.New().Put(httpPrefix + r.serverHost + "/api/beat/pc/name/" + strconv.FormatInt(id, 10)).
		Send(item).
		End()
	if errs != nil {
		r.logger.Info(errs)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		r.logger.Info("response is error:", resp.StatusCode)
		return false
	}
	return true
}

func (r *Server) UpdateConfig(id int64, freq interface{}, conf interface{}) bool {
	item := map[string]interface{}{"freq": freq, "conf": conf}
	resp, body, errs := gorequest.New().Put(httpPrefix + r.serverHost + "/api/beat/pc/config/frequency/" + strconv.FormatInt(id, 10)).
		Send(item).
		End()
	if errs != nil {
		r.logger.Info(errs)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		r.logger.Info(body)
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
	Name     *string `json:"name"`
	Status   string  `json:"status"`
	User     *string `json:"user"`
	Password *string `json:"password"`
	IsValid  bool
	//request *gorequest.SuperAgent
}

type pc struct {
	ID         int64          `json:"id"`
	MacAddress string         `json:"macAddress"`
	IP         string         `json:"ip"`
	Status     string         `json:"status"`
	Name       *string        `json:"name"`
	NetworkID  int64          `json:"networkID"`
	User       *string        `json:"user"`
	Password   *string        `json:"password"`
	DeletedAt  mysql.NullTime `json:"deletedAt"`
}

type Command struct {
	ID     int64  `json:id`
	PcID   int64  `json:pcID`
	IP     string `json:"ip"`
	Status string `json:"status"`
	Type   string `json:"type"`
	Param  string `json:"param"`
}
