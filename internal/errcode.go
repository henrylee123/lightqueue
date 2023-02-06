package lightqueue

type ErrObj struct {
	Code       uint64
	Desc       string
	SucceedNum int64
}

func (e ErrObj) Error() string {
	return e.Desc
}

func Succeed(num int64) ErrObj {
	return ErrObj{Code: 0, Desc: "Succeed", SucceedNum: num}
}

func ZeroErr() ErrObj {
	return ErrObj{Code: 100000, Desc: "Push or Pop zero num"}
}

func QueueIsFull() ErrObj {
	return ErrObj{Code: 100010, Desc: "Queue is full"}
}

func ApplyPushFailed() ErrObj {
	return ErrObj{Code: 100020, Desc: "Applying to push failed"}
}

func UpdateQueueStatusFailed() ErrObj {
	return ErrObj{Code: 100030, Desc: "Update queue status failed"}
}

func QueueIsEmpty() ErrObj {
	return ErrObj{Code: 100040, Desc: "Queue is empty"}
}

func ApplyPopFailed() ErrObj {
	return ErrObj{Code: 100050, Desc: "Applying to pop failed"}
}
