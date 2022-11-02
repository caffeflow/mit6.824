package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"time"
)
import "strconv"

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type HeartArgs struct { //  作用: 请求心跳
	token    string
	taskType int
	time     time.Time
	/*
		 coordinator可以根据taskType和taskId从桶中快速定位到任务, 并结合worker的time来“感受”worker的心跳。
		如果coordinator找不到任务，那么说明worker提交的taskType或taskId存在问题，此时worker发生了错误的心跳。
	*/
}
type HeartReply struct {
	time time.Time
}

type ApplyTaskArgs struct { // 请求任务的参数
	// taskType int
	// token    string
	time time.Time
}

type ApplyTaskReply struct {
	token               string
	taskType            int
	taskId              int
	heartIntervalMinute int
	nReduce             int
	nMap                int
	mapInputFiles       []string // only map worker
	reduceInputFiles    []string // only reduce worker
	time                time.Time
}

type CompleteTaskArgs struct { // 汇报完成任务
	taskType          int
	token             string
	mapOutputFiles    []string // only map worker
	reduceOutputFiles []string // only reduce worker
	time              time.Time
}

type CompleteTaskReply struct {
	time time.Time
}

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
