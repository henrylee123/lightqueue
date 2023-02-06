package internal

import (
	"fmt"
	"go.uber.org/zap"
	"lightqueue/internal/bitmap"
	"runtime"
	"sync/atomic"
	"time"
)

var log, _ = zap.NewProduction()

const queueLen int64 = 32

// 值的类型
type Val interface{}

type Queue struct {
	buf []Val
	// 队头，队尾，队值数
	head, tail, count int64
	p                 Producer
	w                 Worker
	b                 bitmap.Bitmap
	resetSignal       int64
	unblocking        uint32
	blockChan         chan struct{}
	blockNum          int64
}

func (q *Queue) Init() {
	q.buf = make([]Val, queueLen, queueLen)
	q.blockChan = make(chan struct{})
	q.b.Init(uint64(queueLen))
}

func (q *Queue) AddProducer() {
	q.p.Init()
}

func (q *Queue) StopProducer() {
	q.p.Stop()
}

func (q *Queue) AddWorker() {
	q.w.Init()
}

func (q *Queue) StopWorker() {
	q.w.Stop()
}

// get queue free contains
func (q *Queue) Free() int64 {
	size := q.tail - q.head
	if size == 0 {
		if q.count > 0 {
			return 0
		} else {
			return queueLen
		}
	} else if size < 0 {
		return -size
	} else {
		return queueLen - size
	}
}

func (q *Queue) nextPos(pos int64, nextNum int64) int64 {
	return (pos + nextNum) & (queueLen - 1)
}

func (q *Queue) canPush(i int64) bool {
	return !q.b.IsSet(uint64(i))
}

func (q *Queue) waitToCanPush(pos int64) {
	if ok := q.canPush(pos); ok {
		return
	}
	for {
		if ok := q.canPush(pos); ok {
			return
		}
		time.Sleep(time.Duration(100) * time.Nanosecond)
		if ok := q.canPush(pos); ok {
			return
		}
		time.Sleep(time.Duration(100) * time.Millisecond)
		if ok := q.canPush(pos); ok {
			return
		}
		runtime.Gosched()
	}
}

func (q *Queue) BlockPush(elems []Val) {
	if succeed, err := q.Push(elems); !succeed &&
		(err.Code == 100010 || err.Code == 100020) {
		<-q.blockChan
		atomic.AddInt64(&q.blockNum, 1)
	}
}

func (q *Queue) Push(elems []Val) (bool, ErrObj) {
	pushNum := int64(len(elems))
	if pushNum == 0 {
		return false, ZeroErr()
	}

	// 判断是否够容量
	if q.Free() < pushNum {
		return false, QueueIsFull()
	}
	// 申请占有容量
	if ok := q.p.CAStoPush(); !ok {
		return false, ApplyPushFailed()
	}
	// 判断是否够容量
	if q.Free() < pushNum {
		q.p.CASFinishPush()
		return false, QueueIsFull()
	}
	tmpTail := q.tail
	q.tail = q.nextPos(q.tail, pushNum)
	q.p.CASFinishPush()

	// 入队
	for i := int64(0); i < pushNum; i++ {
		pos := q.nextPos(tmpTail, i)
		q.waitToCanPush(pos)
		q.buf[pos] = elems[i]
		q.b.Set(uint64(pos))
		fmt.Println("push: ", pos+1, " head: ", q.head, " tail: ", q.tail)
		fmt.Println(q.b.String())
		atomic.AddInt64(&q.count, 1)
	}

	return true, Succeed(pushNum)
}

func (q *Queue) applyPop(popNum int64) (bool, ErrObj, int64) {
	var ok bool
	ok = q.w.CASToPop()
	if !ok {
		return false, ApplyPopFailed(), 0
	}
	if queueLen-q.Free() < popNum {
		q.w.CASFinishPop()
		return false, QueueIsEmpty(), 0
	}

	tmpHead := q.head
	q.head = q.nextPos(q.head, popNum)
	q.w.CASFinishPop()

	return true, ErrObj{}, tmpHead
}

func (q *Queue) canPop(i int64) bool {
	return q.b.IsSet(uint64(i))
}

func (q *Queue) waitToCanPop(pos int64) (dontWait bool) {
	if ok := q.canPop(pos); ok {
		return
	}

	var retryTimes int
	for {
		time.Sleep(time.Duration(150) * time.Millisecond)
		if ok := q.canPop(pos); ok {
			return
		}
		//runtime.Gosched()

		if retryTimes > 3 {
			if q.p.CheckAllStop() {
				q.ResetQueue()
				dontWait = true
				return
			}
			if ok := q.canPop(pos); ok {
				return
			}
			time.Sleep(time.Duration(300) * time.Millisecond)
		}
		retryTimes++
	}
}

func (q *Queue) ResetQueue() {
	atomic.AddInt64(&q.resetSignal, 1)
	if q.resetSignal == q.w.Count() {
		q.b.Reset(uint64(queueLen))
		q.head = q.tail
		q.count = 0
		q.resetSignal = 0
	}
}

func (q *Queue) Pop(popNum int64, receiver *[]Val) (bool, ErrObj) {
	if popNum <= 0 {
		return false, ZeroErr()
	}
	if queueLen-q.Free() < popNum {
		return false, QueueIsEmpty()
	}

	ok, err, tmpHead := q.applyPop(popNum)
	if !ok {
		return false, err
	}

	// pop
	for i := int64(0); i < popNum; i++ {
		pos := q.nextPos(tmpHead, i)
		if dontWait := q.waitToCanPop(pos); dontWait {
			return true, Succeed(i)
		}
		q.b.Unset(uint64(pos))
		fmt.Println("pop: ", pos+1, " head: ", q.head, " tail: ", q.tail)
		fmt.Println(q.b.String())
		*receiver = append(*receiver, q.buf[pos])
		atomic.AddInt64(&q.count, -1)
	}

	// active block push worker
	if q.blockNum != 0 {
		if q.Free() > queueLen/2 {
			if ok := q.CAStoUnblock(); ok && q.Free() > queueLen/2 {
				<-q.blockChan
				atomic.AddInt64(&q.blockNum, -1)
				q.CASUnblocked()
			}
		}
	}

	return true, Succeed(popNum)
}

func (q *Queue) CAStoUnblock() bool {
	return atomic.CompareAndSwapUint32(&q.unblocking, 0, 1)
}

func (q *Queue) CASUnblocked() bool {
	return atomic.CompareAndSwapUint32(&q.unblocking, 1, 0)
}
