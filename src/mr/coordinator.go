package mr

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)
import "net"
import "net/rpc"
import "net/http"

/*
coordinator.go 整体规划

1. `MapTask` 结构体表示了Map任务的各种信息，通过 `MakeCoordinator`方法 传入文件作为参数，这些文件就对应构建了多个map任务对象，
这些任务对象被存放在 `coordinator struct` 的 `[]MapTask` 任务桶中。

2. `Worker`进程 可以远程调用 `coordinator.go` 中的 `HeartOrApply` 方法，来申请任务。

3. 同样地，当mapWorker完成后，我们会构建更多的reduce任务队，存储在任务桶中，等待worker进程来申请任务。

4. 当worker进程完成任务后，它应该远程调用 `coordinator.go`中的 `CompleteTask` 方法 来告知输出文件存放的位置，等等，之后接着就会涉及进程关闭等行动。


*/

const ( // task type
	mapType int = iota
	reduceType
)
const ( // task status ---- status=idle,表示worker可以拿到该task进行执行,此时更新status=inProgress,如果worker完成了task,那么status=completed, 当worker出现一些问题时,我们可以重置status=idle来让新的worker来执行它.
	idleStatus int = iota
	inProgressStatus
	completeStatus
)

type Task struct {
	taskType   int
	taskId     int
	inputFiles []string
	numReduce  int
	/*
		inputFiles说明：
			mapTask 的输入只有一个文件。如果文件过大可以切分多个文件，如果小文件很多就合并文件，这是创建mapTask之前应该做。
		outputFiles说明：
			mapTask 的输出有numReduce个中间文件，文件命名格式 m-X-Y, X是mapTaskId,Y是reduceTaskId
			reduceTask 的输出有一个文件, 文件命名格式 mr-out-X, X是reduceTaskId

		共享文件系统:
			所有worker,包括 map task 和reduce task 应该共享文件系统, 每个mapper通过rpc拿到文件并处理后,会输出中间文件在自己的目录下, reducer会去每个mapper的目录下去查找这些中间文件.
			在本实验中单机分布式场景,仅仅基于linux文件系统即满足共享文件系统的条件, 但在完全分布式场景下, 应该使用全局文件系统,诸如GFS或HDFS.
	*/
}

type BindInfo struct {
	// 当worker申请到task后，BindInfo对象被coordinator作为调度管理的单元，当worker失联时，coordinator可以直接删除对应的bindInfo对象。
	token                 string // 唯一值, 在创建该BindInfo对象(worker申请到task时)动态生成。
	task                  *Task
	taskStatus            int
	outputFiles           []string
	recentWorkerHeartTime time.Time // coordinator感知到worker心跳的时间。 如果心跳太久了，coordinator会假定worker失去联系，会重置对应的task为idle状态。
	heartIntervalMinute   int
}

type BindInfoManager struct {
	sync.RWMutex
	m map[string]*BindInfo
}

func (b *BindInfoManager) Get(token string) (*BindInfo, bool) {
	b.RLock()
	info, ok := b.m[token]
	b.RUnlock()
	return info, ok
}

func (b *BindInfoManager) Put(token string, bindInfo *BindInfo) {
	b.Lock()
	b.m[token] = bindInfo
	b.Unlock()
}

func (b *BindInfoManager) Remove(token string) {
	b.Lock()
	delete(b.m, token)
	b.Unlock()
}

func (b *BindInfoManager) Range(f func(idx int, key string, value *BindInfo) bool) {
	b.RLock()
	idx := 0
	for k, v := range b.m {
		if !f(idx, k, v) { // 执行f
			break
		}
		idx += 1
	}
	b.RUnlock()
}

func (b *BindInfoManager) Size() int {
	b.RLock()
	size := len(b.m)
	b.RUnlock()
	return size
}

type Coordinator struct {
	// coordinator meta info
	numReduce           int      // 取决于任务如何设置key的种类
	numMap              int      // 取决于最终有多少文件
	files               []string // 客户端发送的文件地址
	heartIntervalMinute int      // worker心跳间隔
	mu                  sync.RWMutex
	// idle chan
	idleChanMap    chan *Task
	idleChanReduce chan *Task
	// inProcess bind
	inProcessBindMap    *BindInfoManager
	inProcessBindReduce *BindInfoManager
	// complete bind
	completeBindMap    *BindInfoManager
	completeBindReduce *BindInfoManager
	// kill
	killKeepALiveThread   bool
	killCoordinatorThread bool
}

// Your code here -- RPC handlers for the worker to call.

func (c *Coordinator) startKeepLiveThread() {
	/*
		启动新线程，周期性地检查worker心跳。  遇到Kill信号则退出循环，线程自动结束。
	*/
	checkALive := func(inProcessBindManager *BindInfoManager, intervalMinute int) {
		for true {
			if c.killKeepALiveThread { // 退出循环
				fmt.Printf("keepALive 线程运行结束\n")
				break
			}
			if inProcessBindManager.Size() > 0 {
				legalTime := time.Now().Add(-time.Duration(int64(time.Minute) * int64(intervalMinute)))
				notLiveTokenCh := make(chan string)
				findNotLive := func(idx int, token string, bind *BindInfo) bool {
					if bind.recentWorkerHeartTime.Before(legalTime) {
						notLiveTokenCh <- bind.token // 不能直接删除该bindInfo，因为BindInfoManger的读锁和写锁嵌套会造成死锁
					}
					return true
				}
				inProcessBindManager.Range(findNotLive)
				for token := range notLiveTokenCh {
					bindInfo, _ := inProcessBindManager.Get(token)
					task := bindInfo.task
					if task.taskType == mapType { // idle管道 <- task
						c.idleChanMap <- task
					} else {
						c.idleChanReduce <- task
					}
					inProcessBindManager.Remove(token)
					fmt.Printf("worker心跳过期,已重置任务！ token=%v, taskType=%v, taskId=%v\n", token, task.taskType, task.taskId)
				}
			}
			time.Sleep(time.Duration(int64(time.Minute) * int64(intervalMinute/2)))
		}
	}

	go checkALive(c.inProcessBindMap, c.heartIntervalMinute)
	go checkALive(c.inProcessBindReduce, c.heartIntervalMinute)
}

func (c *Coordinator) genWorkerToken() string {
	/*
		该方法非RPC  -  新分配一个任务时，coordinator会生成一个token来唯一标识worker的身份。
		1、coordinator需要维护token与任务桶下标的映射关系, 以绑定worker与task.
		2、worker应该记住这个token，每次调用RPC时发送这个token给coordinator来表明自己的身份。
	*/
	t := time.Now().Unix()
	h := md5.New()
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(t))
	h.Write([]byte(b))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Coordinator) DoHeart(args *HeartArgs, reply *HeartReply) error {
	//token string
	var bindManager *BindInfoManager
	if args.taskType == mapType {
		bindManager = c.inProcessBindMap
	} else if args.taskType == reduceType {
		bindManager = c.inProcessBindReduce
	} else {
		return errors.New("taskType无效，心跳失败")
	}
	bindInfo, ok := bindManager.Get(args.token)
	if !ok {
		return errors.New("token无效，心跳失败")
	}
	bindInfo.recentWorkerHeartTime = time.Now() // 记录心跳
	reply.time = time.Now()
	return nil
}

func (c *Coordinator) BuildMapPhase() error {
	for i, file := range c.files {
		c.idleChanMap <- &Task{
			taskType:   mapType,
			taskId:     i,
			inputFiles: []string{file},
			numReduce:  c.numReduce,
		}
	}
	return nil
}

func (c *Coordinator) BuildReducePhase() error {
	numCompleteMap := c.completeBindMap.Size()
	if numCompleteMap < c.numMap {
		return errors.New(fmt.Sprintf("map任务全部完成才能开启reduce阶段，目前map任务完成进度= %v/%v", numCompleteMap, c.numMap))
	}
	// shuffle
	reduceInputFiles := map[int][]string{} // reduceId : files

	tmpFunc := func(mapId int, token string, mapBind *BindInfo) bool {
		for reduceId := 0; reduceId < c.numReduce; reduceId++ {
			reduceInputFiles[reduceId] = append(reduceInputFiles[reduceId], mapBind.outputFiles[reduceId])
		}
		return true
	}
	c.completeBindMap.Range(tmpFunc)

	for i := 0; i < c.numReduce; i++ {
		c.idleChanReduce <- &Task{
			taskType:   reduceType,
			taskId:     i,
			inputFiles: reduceInputFiles[i],
			numReduce:  c.numReduce,
		}
	}
	return nil
}

func (c *Coordinator) ApplyTask(args *ApplyTaskArgs, reply *ApplyTaskReply) error {
	fmt.Printf("worekr try to apply a task by coordinator's RPC: ")
	var task *Task
	var bindManager *BindInfoManager
	// 尝试获取任务
	task = <-c.idleChanMap
	bindManager = c.inProcessBindMap
	if task == nil {
		task = <-c.idleChanReduce
		bindManager = c.inProcessBindReduce
	}
	if task == nil {
		if c.completeBindReduce.Size() == c.numMap && c.completeBindMap.Size() == c.numReduce {
			if len(c.idleChanMap) != 0 || len(c.idleChanReduce) != 0 || c.inProcessBindMap.Size() != 0 || c.inProcessBindReduce.Size() != 0 {
				return errors.New("coordinator调度管理存在问题")
			}
			return errors.New("任务全部完成！")
		}
		return errors.New("暂无可申请的任务，待会再试一试!")
	}
	// add new bindInfo to bindManager
	token := c.genWorkerToken() // 生成workerToken
	bindInfo := &BindInfo{
		token:                 token,
		task:                  task,
		taskStatus:            inProgressStatus,
		outputFiles:           nil,
		recentWorkerHeartTime: time.Now(),
		heartIntervalMinute:   c.heartIntervalMinute,
	}
	bindManager.Put(token, bindInfo)
	// reply
	reply.token = c.genWorkerToken()
	reply.taskId = task.taskId
	reply.time = time.Now()
	reply.heartIntervalMinute = c.heartIntervalMinute
	reply.taskType = task.taskType
	reply.nReduce = c.numReduce
	reply.nMap = c.numMap
	if task.taskType == reduceType {
		reply.reduceInputFiles = task.inputFiles
	} else {
		reply.mapInputFiles = task.inputFiles
	}
	fmt.Printf("发送任务成功！ taskType=%v, taskId=%v, workerToken=%v", task.taskType, task.taskId, token)

	return nil
}

func (c *Coordinator) CompleteTask(args *CompleteTaskArgs, reply *CompleteTaskReply) error {
	if args.taskType != mapType || args.taskType != reduceType {
		return errors.New(fmt.Sprintf("taskType不正确! taskType=%v", args.taskType))
	}
	var inProcessBindManager *BindInfoManager
	var completeBindManager *BindInfoManager
	var outputFiles []string
	if args.taskType == mapType {
		inProcessBindManager = c.inProcessBindMap
		completeBindManager = c.completeBindMap
		outputFiles = args.mapOutputFiles
	} else {
		inProcessBindManager = c.inProcessBindReduce
		completeBindManager = c.completeBindReduce
		outputFiles = args.reduceOutputFiles
	}
	bindInfo, ok := inProcessBindManager.Get(args.token)
	if ok {
		inProcessBindManager.Remove(args.token)
		completeBindManager.Put(args.token, bindInfo)
		bindInfo.recentWorkerHeartTime = time.Now()
		bindInfo.taskStatus = completeStatus
		bindInfo.outputFiles = outputFiles
	} else {
		return errors.New(fmt.Sprintf("token不正确! token=%v", args.token))
	}
	reply.time = time.Now()

	// 推动coordinator到新的阶段
	if c.completeBindMap.Size() == c.numMap && c.completeBindReduce.Size() == 0 { // map 任务全部完成, reduce 阶段尚未建立
		fmt.Printf("map阶段完成, 正在构建reduce阶段的任务\n")
		c.BuildReducePhase()
	} else if c.completeBindReduce.Size() == c.numReduce { // reduce 任务全部完成
		fmt.Printf("reduce阶段完成，输出文件如下:")
		printReduceOutPutFiles := func(idx int, token string, bindInfo *BindInfo) bool {
			fmt.Printf("%v\n", bindInfo.outputFiles[0]) // 每个reduce task只有一个输出文件
			return true
		}
		c.completeBindReduce.Range(printReduceOutPutFiles)
		fmt.Printf("准备结束Coordinator!")
		c.killKeepALiveThread = true
		c.killCoordinatorThread = true
	}

	return nil
}

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)

}

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//

func (c *Coordinator) Done() bool {
	ret := false
	// Your code here.
	ret = c.killCoordinatorThread
	return ret
}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	// Your code here.
	if files == nil || len(files) < 1 {
		panic("files为nil或无文件")
	}
	c := Coordinator{
		numReduce:           nReduce,
		numMap:              len(files),
		files:               files,
		heartIntervalMinute: 5,
		idleChanMap:         make(chan *Task),
		idleChanReduce:      make(chan *Task),
		//inProcessBindMap: new(WorkTaskBind),
		inProcessBindMap:      &BindInfoManager{m: map[string]*BindInfo{}},
		inProcessBindReduce:   &BindInfoManager{m: map[string]*BindInfo{}},
		completeBindMap:       &BindInfoManager{m: map[string]*BindInfo{}},
		completeBindReduce:    &BindInfoManager{m: map[string]*BindInfo{}},
		killCoordinatorThread: false,
		killKeepALiveThread:   false,
	}
	c.server()
	return &c
}
