package bkloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/nomadit/antminerbeat/beat/db"
	"github.com/nomadit/antminerbeat/beat/digestRequest"
	"log"
)

type Status string

type CommandType string

const (
	Run    Status = "RUN"
	Finish Status = "FINISH"
	Error  Status = "ERROR"
)

type Command struct {
	db.Command
	user     *string
	password *string
}

const (
	changeName    = "CHANG_HOSTNAME"
	changeConfig  = "CHANG_CONFIGURE"
)

func (c *Command) run() error {
	switch c.Type {
	case changeName:
		if err := c.changeName(); err != nil {
			c.Status = string(Error)
			return err
		}
	case changeConfig:
		if err := c.changeConfigList(); err != nil {
			c.Status = string(Error)
			return err
		}
	}
	c.Status = string(Finish)
	return nil
}
func (c *Command) changeConfigList() error {
	var params paramType
	err := json.Unmarshal([]byte(c.Param), &params)
	if err != nil {
		log.Println(err)
		return err
	}

	freqParam := fmt.Sprintf("_ant_freq=%d", params.Freq)
	w := bytes.Buffer{}
	for i, conf := range *params.Conf {
		w.WriteString(fmt.Sprintf("_ant_pool%durl=%s&_ant_pool%duser=%s&_ant_pool%dpw=%s&",
			i+1, conf.URL, i+1, conf.Wallet, i+1, *conf.Password))
		log.Println(w.String())
	}
	if len(*params.Conf) < 3 {
		for i := len(*params.Conf); i < 3; i++ {
			w.WriteString(fmt.Sprintf("_ant_pool%durl=%s&_ant_pool%duser=%s&_ant_pool%dpw=%s&",
				i+1, "", i+1, "", i+1, ""))
			log.Println(w.String())
		}
	}
	uri := "/cgi-bin/set_miner_conf.cgi"
	hostTemplate := "http://%s"
	host := fmt.Sprintf(hostTemplate, c.IP)
	bodyStr := fmt.Sprintf("%s_ant_nobeeper=&_ant_notempoverctrl="+
		"&_ant_fan_customize_switch=&_ant_fan_customize_value=&%s",
		w.String(), freqParam)
	body := []byte(bodyStr)

	resp, err := digestRequest.Digest("POST", host, uri, *c.user, *c.password, &body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *Command) changeName() error {
	var params paramType
	err := json.Unmarshal([]byte(c.Param), &params)
	if err != nil {
		log.Println(err)
		return err
	}

	w := bytes.Buffer{}
	w.WriteString(fmt.Sprintf("_ant_conf_nettype=DHCP&_ant_conf_hostname=%s", params.Hostname))
	w.WriteString("&_ant_conf_ipaddress=&_ant_conf_netmask=&_ant_conf_gateway=&_ant_conf_dnsservers=")
	uri := "/cgi-bin/set_network_conf.cgi"
	hostTemplate := "http://%s"
	host := fmt.Sprintf(hostTemplate, c.IP)
	body := w.Bytes()

	resp, err := digestRequest.Digest("POST", host, uri, *c.user, *c.password, &body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

type PoolConfig struct {
	URL      string  `json:"url"`
	Wallet   string  `json:"wallet"`
	Password *string `json:"password"`
}
type paramType struct {
	Freq     int           `json:"freq"`
	Conf     *[]PoolConfig `json:"conf"`
	Hostname string        `json:"hostname"`
}
