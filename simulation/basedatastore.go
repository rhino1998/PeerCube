package simulation

import (
	"sync"
)

type BaseDataStore struct {
	data  map[string]string
	mutex sync.RWMutex
}

func (d *BaseDataStore) Get(key string) string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data[key]
}

func (d *BaseDataStore) Set(key string, data string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.data[key] = data
}

func (d *BaseDataStore) Delete(key string) string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	v := d.data[key]
	delete(d.data, key)
	return v
}
