package db

type Scan interface {
	Start() error
	Stop() error
	Add(m *Miner)
	Remove(mac string )
}
