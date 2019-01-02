package db

import (
	"github.com/pkg/errors"
	"sync"
)

var IpTable = ipTable{}

type ipTable struct {
	macIPTable sync.Map
}

func (t *ipTable) clear()  {
	t.macIPTable.Range(func(key, value interface{}) bool {
		t.macIPTable.Delete(key)
		return true
	})
}

func (t *ipTable) SetTable(miners *[]Miner) {
	for _, row := range *miners {
		t.macIPTable.Store(row.Mac, row)
	}
}

func (t *ipTable) Find(key string) (value interface{}, ok bool) {
	return t.macIPTable.Load(key)
}

func (t *ipTable) Set(key string, value Miner)  {
	t.macIPTable.Store(key, value)
}

func (t *ipTable) SetInValid(key string, invalid bool) error {
	if val, ok := t.macIPTable.Load(key); ok {
		m := val.(Miner)
		m.IsValid = invalid
		t.macIPTable.Store(key, m)
		return nil
	} else {
		return errors.New("not found key:" + key)
	}
}

func (t *ipTable) GetValidList() *[]Miner {
	var list []Miner
	t.macIPTable.Range(func(key, value interface{}) bool {
		m := value.(Miner)
		if m.IsValid {
			list = append(list, m)
		}
		return true
	})
	return &list
}
