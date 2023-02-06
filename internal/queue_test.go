package lightqueue

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"net/http"
	_ "net/http/pprof"
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	q := Queue{}
	q.Init()
	var ok bool
	var err ErrObj

	var res = make([]Val, 0, 1000)

	q.AddWorker()
	ok, err = q.Pop(1, &res)
	fmt.Println(res, ok, err, q.b)
	assert.Equal(t, res, []Val{}, "pop0 err")

	q.AddProducer()
	ok, err = q.Push([]Val{"a", "b", "c", "d"})
	if !ok {
		fmt.Println(err.Error())
	}

	ok, err = q.Pop(1, &res)
	fmt.Println(res, ok, err, q.b)
	assert.Equal(t, res, []Val{"a"}, "pop1 err")

	ok, err = q.Pop(2, &res)
	fmt.Println(res, ok, err, q.b)
	assert.Equal(t, res, []Val{"a", "b", "c"}, "pop2 err")

	ok, err = q.Pop(3, &res)
	fmt.Println(res, ok, err, q.b)
	assert.Equal(t, res, []Val{"a", "b", "c"}, "pop3 err")
}

func TestMultiplyWorkers(t *testing.T) {
	go func() {
		http.ListenAndServe("0.0.0.0:8000", nil)
	}()

	q := Queue{}
	q.Init()
	eg := errgroup.Group{}

	var res = make([]Val, 0, 100000)
	eg.Go(func() error {
		q.AddWorker()
		for {
			for i := 0; i < 100; i++ {
				ok, err := q.Pop(3, &res)
				if !ok && err.Code == 100040 {
					time.Sleep(time.Nanosecond * 50)
					continue
				}
			}
			time.Sleep(time.Second)
			ok, err := q.Pop(3, &res)
			if !ok && err.Code == 100040 {
				return nil
			}
		}
	})

	var p int64
	var np int64

	q.AddProducer()
	for i := 0; i < 10000; i++ {
		ok, err := q.BPush([]Val{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		if ok {
			p += 1
			//time.Sleep(time.Millisecond)
		} else {
			np += 1
			fmt.Println(err.Error())
		}
	}
	fmt.Println("finish pushed")

	err := eg.Wait()
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Printf("pushed: %d poped: %d not pushed: %d", p*10, len(res), np*10)
}
