package ifconfig

import (
	"fmt"
	"log"
	"net"
	"strconv"
)


func Networks() *[]Ifaddr {
	ifaces, err := net.Interfaces()
	if err != nil {
		fmt.Println(err)
	}
	// handle err
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			fmt.Println(err)
		}
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.To4() != nil && len(i.HardwareAddr.String()) > 0 {
				ones, _ := ip.DefaultMask().Size()
				aIfnet.addLocalInfo(i.HardwareAddr.String(),
					ip.String(), ip.String()+"/"+strconv.Itoa(ones))
			}
		}
	}
	networkList := []Ifaddr{}
	for _, addr := range *aIfnet {
		networkList = append(networkList, addr)
	}
	return &networkList
}

var aIfnet = newIfnet()

type ifnet map[string]Ifaddr

type Ifaddr struct {
	Mac       string
	Ip        string
	Network   string
}

func newIfnet() *ifnet {
	return &ifnet{}
}

func (r *ifnet) printLocalInfoes() {
	for key, value := range *r {
		log.Println(key, value.Ip, value.Network)
	}
}

func (r *ifnet) addLocalInfo(mac string, ip string, network string) {
	if _, found := (*r)[mac]; found {
		return
	}
	(*r)[mac] = Ifaddr{Mac: mac, Ip: ip, Network: network}
}
