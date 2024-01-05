package structure

import (
	"sync"
	"sync/atomic"
)

type InfoCache struct {
	info map[uintptr]Info `exhaustruct:"optional"`
	mu   sync.RWMutex     `exhaustruct:"optional"`

	Hit  atomic.Int64 `exhaustruct:"optional"`
	Miss atomic.Int64 `exhaustruct:"optional"`
}

func (c *InfoCache) Get(id uintptr, fn func() (Info, error)) (Info, error) {
	c.mu.RLock()
	info, ok := c.info[id]
	c.mu.RUnlock()

	if ok {
		c.Hit.Add(1)
		return info, nil
	}

	c.Miss.Add(1)
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.info == nil {
		c.info = make(map[uintptr]Info)
	}

	info, err := fn()
	if err != nil {
		return info, err
	}

	c.info[id] = info

	return info, nil
}
