package lightqueue

import asyncLog "github.com/ibbd-dev/go-async-log"

var logger = asyncLog.NewLevelLog("./info-log.log", asyncLog.LevelInfo) // 只有Info级别或者以上级别的日志才会被记录
