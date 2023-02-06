package lightqueue

import (
	"github.com/petermattis/goid"
	"sync"
	"sync/atomic"
)

type Producer struct {
	applyPop uint32
	gids     []int64
	initLock sync.Mutex
}

func (p *Producer) Init() {
	p.initLock.Lock()
	defer p.initLock.Unlock()

	cGid := goid.Get()
	for _, gid := range p.gids {
		if cGid == gid {
			return
		}
	}
	if p.gids == nil {
		p.gids = []int64{cGid}
	} else {
		p.gids = append(p.gids, cGid)
	}
}

func (p *Producer) Stop() {
	p.initLock.Lock()
	defer p.initLock.Unlock()
	cGid := goid.Get()
	if p.gids != nil {
		for i, gid := range p.gids {
			if cGid == gid {
				p.gids = append(p.gids[:i], p.gids[i+1:]...)
				return
			}
		}
	}
}

func (p *Producer) CAStoPush() bool {
	return atomic.CompareAndSwapUint32(&p.applyPop, 0, 1)
}

func (p *Producer) CASFinishPush() bool {
	return atomic.CompareAndSwapUint32(&p.applyPop, 1, 0)
}

func (p *Producer) CheckAllStop() bool {
	if len(p.gids) == 0 {
		return true
	}
	return false
}

func (p *Producer) Num() int64 {
	return int64(len(p.gids))
}
