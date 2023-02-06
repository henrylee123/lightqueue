package internal

import (
	"github.com/petermattis/goid"
	"sync"
	"sync/atomic"
)

type Worker struct {
	applyPop uint32
	initLock sync.Mutex
	gids     []int64
}

func (w *Worker) Init() {
	w.initLock.Lock()
	defer w.initLock.Unlock()

	cGid := goid.Get()
	for _, gid := range w.gids {
		if cGid == gid {
			return
		}
	}
	if w.gids == nil {
		w.gids = []int64{cGid}
	} else {
		w.gids = append(w.gids, cGid)
	}
}

func (w *Worker) Stop() {
	w.initLock.Lock()
	defer w.initLock.Unlock()
	cGid := goid.Get()
	if w.gids != nil {
		for i, gid := range w.gids {
			if cGid == gid {
				w.gids = append(w.gids[:i], w.gids[i+1:]...)
				return
			}
		}
	}
}

func (w *Worker) CASToPop() bool {
	return atomic.CompareAndSwapUint32(&w.applyPop, 0, 1)
}

func (w *Worker) CASFinishPop() bool {
	return atomic.CompareAndSwapUint32(&w.applyPop, 1, 0)
}

func (w *Worker) Count() int64 {
	return int64(len(w.gids))
}
