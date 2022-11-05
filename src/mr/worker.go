package mr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"
)
import "log"
import "net/rpc"
import "hash/fnv"

// for sorting by key.
type ByKey []KeyValue

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//

type Bind struct {
	workerToken         string
	taskType            int
	taskId              int
	heartIntervalMinute int
	files               []string
	nReduce             int
	nMap                int
}

func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {

	// Your worker implementation here.

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

	// 定期向coordinator申请任务
	for {
		applyTaskArgs := &ApplyTaskArgs{
			time: time.Now(),
		}
		applyTaskReply := &ApplyTaskReply{}
		ok := call("Coordinator.ApplyTask", applyTaskArgs, applyTaskReply)
		if !ok {
			break
		}
		if ok && applyTaskReply.token == "" { // 响应，但是申请失败
			fmt.Printf("call true, but apply false, 稍后自动再试一试\n")
			time.Sleep(time.Minute * 1)
			continue
		}
		// 绑定任务
		bind := Bind{
			workerToken:         applyTaskReply.token,
			taskType:            applyTaskReply.taskType,
			taskId:              applyTaskReply.taskId,
			heartIntervalMinute: applyTaskReply.heartIntervalMinute,
			files:               applyTaskReply.mapInputFiles,
			nReduce:             applyTaskReply.nReduce,
			nMap:                applyTaskReply.nMap,
		}

		// 异步开启心跳
		heartArgs := &HeartArgs{token: bind.workerToken, taskType: bind.taskType, time: time.Now()}
		heartReply := &HeartReply{}
		go call("coordinator.CompleteTask", heartArgs, heartReply)
		go func() {
			for {
				time.Sleep(time.Duration(bind.heartIntervalMinute / 4))
				if time.Now().After(heartReply.time.Add(time.Duration(int64(time.Minute) * int64(bind.heartIntervalMinute)))) {
					fmt.Printf("心跳预期 token=%v", bind.workerToken)
				}
			}
		}()

		if bind.taskType == mapType { // map task
			filename := bind.files[0] //
			intermediate := []KeyValue{}
			file, err := os.Open(filename)
			if err != nil {
				log.Fatalf("cannot open %v", filename)
			}
			content, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatalf("cannot read %v", filename)
			}
			file.Close()
			kva := mapf(filename, string(content)) // mapf参数: 文件名;文件的内容

			intermediate = append(intermediate, kva...)

			// 按key分桶
			buckets := make([][]KeyValue, bind.nReduce)
			for i := range buckets {
				buckets[i] = []KeyValue{}
			}
			for _, kva := range intermediate {
				i := ihash(kva.Key) % bind.nReduce
				buckets[i] = append(buckets[i], kva)
			}
			for i := range buckets { // 桶内排序
				sort.Sort(ByKey(buckets[i]))
			}
			ofilenames := make([]string, bind.nReduce)
			for i := range buckets {
				oname := "mr-" + string(bind.taskId) + "-" + string(i) // i 表示 Reducer 的编号
				ofile, _ := ioutil.TempFile("", oname+"_temp")
				encode := json.NewEncoder(ofile)
				for _, kva := range buckets {
					err := encode.Encode(&kva)
					if err != nil {
						fmt.Printf("不能写入 %v", oname)
					}
				}
				os.Rename(ofile.Name(), oname)
				ofilenames = append(ofilenames, oname)
				ofile.Close()
			}

			// RPC 向coordinator提交map任务
			completeTaskArgs := &CompleteTaskArgs{
				taskType:          bind.taskType,
				token:             bind.workerToken,
				mapOutputFiles:    ofilenames,
				reduceOutputFiles: nil,
				time:              time.Now(),
			}
			completeTaskReply := &CompleteTaskReply{}
			ok := call("coordinator.CompleteTask", completeTaskArgs, completeTaskReply)
			if ok {
				fmt.Printf("coordinator 已接受到提交的信息 于时间: %v", completeTaskReply.time)
				break
			}
		} else { // reduce task
			intermediate := []KeyValue{}
			for _, iname := range bind.files {
				file, err := os.Open(iname)
				if err != nil {
					fmt.Printf("不能打开文件 %v", file)
				}
				decode := json.NewDecoder(file)
				for {
					var kv KeyValue
					if err := decode.Decode(&kv); err != nil {
						fmt.Printf("json无法解析文件内容 %v", err)
						break
					}
					intermediate = append(intermediate, kv)
				}

				file.Close()
			}
			sort.Sort(ByKey(intermediate))
			// 输出最终文件
			oname := "mr-out-" + string(bind.taskId)
			ofiles := []string{}
			ofiles = append(ofiles, oname)
			ofile, _ := ioutil.TempFile("", oname+"_temp")
			i := 0
			for i < len(intermediate) {
				j := i + 1
				for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
					j++
				}
				values := []string{}
				for k := i; k < j; k++ {
					values = append(values, intermediate[k].Value)
				}
				output := reducef(intermediate[i].Key, values)
				fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output) // 输出到文件

				i = j
			}
			os.Rename(ofile.Name(), oname)
			ofile.Close()
			// 删除中间文件
			for _, iname := range bind.files {
				if err := os.Remove(iname); err != nil {
					fmt.Printf("不能删除中间文件 %v", iname)
				}
			}

			// RPC - 向coordinator提交最终结果
			completeTaskArgs := &CompleteTaskArgs{
				taskType:          bind.taskType,
				token:             bind.workerToken,
				mapOutputFiles:    nil,
				reduceOutputFiles: ofiles,
				time:              time.Now(),
			}
			completeTaskReply := &CompleteTaskReply{}
			ok := call("Coordinator.CompleteTask", completeTaskArgs, completeTaskReply)
			if ok {
				fmt.Printf("reduce task 提交成功, coordinator接受时间: %v", completeTaskReply.time)
			}

		}
		time.Sleep(time.Second * 10) // 继续尝试申请任务
	}
	return
}

//
// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
