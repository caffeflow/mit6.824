[6.824 Schedule: Spring 2022 (mit.edu)](https://pdos.csail.mit.edu/6.824/schedule.html)

# basic

- go 1.17.6
- Linux

# lab 1 MapReduce

[**LEC 1:** ](https://pdos.csail.mit.edu/6.824/papers/mapreduce.pdf)[Introduction](https://pdos.csail.mit.edu/6.824/notes/l01.txt), [video](https://youtu.be/WtZ7pcRSkOA)
**Preparation:** **Read** [MapReduce (2004)](https://pdos.csail.mit.edu/6.824/papers/mapreduce.pdf)
**Assigned:** [Lab 1: MapReduce](https://pdos.csail.mit.edu/6.824/labs/lab-mr.html)

官方提供了单线程版mr作为参考代码，运行方式如下

```bash
$ cd ~/6.824
$ cd src/main
$ go build -race -buildmode=plugin ../mrapps/wc.go
$ rm mr-out*
$ go run -race mrsequential.go wc.so pg*.txt
$ more mr-out-0
A 509
ABOUT 2
ACT 8
...
```

下面是lab1中涉及到的一些文件

**Reference Code：1. src/main/mr/***；2. src/mrapps/wc.go

**Implementation Code：** 1. src/mr/coordinator.go； 2. src/mr/rpc.go；3、src/mr/worker.go

**test script**  这个测试文件如下，来评估lab1的实现结果。

```bash
$ cd ~/6.824/src/main
$ bash test-mr.sh
*** Starting wc test.
```

## 翻译

### Introduction

在这个实验室中，你将建立一个MapReduce系统。
你将实现一个调用应用程序Map和Reduce函数并处理读写文件的worker进程，以及一个向worker分派任务并处理失败worker的coordinator进程。
你将构建类似于MapReduce论文的东西。(注意：本实验室使用 "coordinator"而不是论文中的 "master"）。

### Getting started

你需要用Go语言去做实验。使用git（版本控制系统）获取初始软件版本。

```bash
$ git clone git://g.csail.mit.edu/6.824-golabs-2022 6.824
$ cd 6.824
$ ls
Makefile src
$
```

我们在src/main/mrsequential.go中为您提供了一个简单的顺序mapreduce实现。它在单个进程中执行Maps和Reduces。我们还为您提供了几个MapReduce
应用程序：mrapps/wc.go中单词计数。和mrapps/indexer.go中的文本索引器。您可以按如下顺序运行单词计数：

```bash
$ cd ~/6.824
$ cd src/main
$ go build -race -buildmode=plugin ../mrapps/wc.go
$ rm mr-out*
$ go run -race mrsequential.go wc.so pg*.txt
$ more mr-out-0
A 509
ABOUT 2
ACT 8
...
```

（注意：-race启用Go的竞争检测器。我们建议您使用竞争检测器开发和测试6.824实验室代码。当我们为您的实验室评分时，我们不会使用竞争检测器。然而，如果您的
代码有竞争，即使没有进制检测器，也很有可能在测试时失败。）

mrsequential.go输出文件是mr-out-0,输入来自名为pg-xxx.txt的文本文件。
请随意从mrsequential.go借用代码。你还应该看看mrapps/wc,查看MapReduce应用程序代码。

### Your Job

您的工作是实现一个分布式MapReduce，它由两个程序组成，一个是the coordinator，另一个是the worker。只有一个coordinator process和一个或多个worker process并行执行。在一个真实的
系统中，工人们将在一堆不同的机器上运行，但对于这个实验室，你将在一台机器上运行他们。the workers 将通过 RPC 与 coordinator 交谈。每个worker process将向 coordinator process 请求任务，从一
个或多个文件读取任务的输入，执行任务，并将任务的输出写入一个或更多文件。the coordinator 应该注意一个 worker 是否在合理的时间内完成了任务（对于这个实验室，用10秒），
并将相同的任务交给另一个工人。

我们给了你一点代码来开始。the coordinator 和 worker 的 “main” 例行程序在 main/mrcoordinator.go 和 main/mrworker.go 。不要更改这些文件。
您应该将您的实现放在mr/coordinator.go，mr/worker.go, mr/rpc.go。

下面是如何在单词计数MapReduce应用程序上运行代码。首先，确保单词计数插件是可以被构建的：

```bash
$ go build -race -buildmode=plugin ../mrapps/wc.go
```

在"main"目录中，运行 coordinator.

```bash
$ rm mr-out*
$ go run -race mrcoordinator.go pg-*.txt
```

pg-*.txt 是 mrcoordinator.go的输入参数，它表示输入的文件们；每个文件对应于一个“split”，并且是一个Map任务的输入。

在一个或多个执行窗口中，运行一些workers:

```bash
$ go run -race mrworker.go wc.so
```

当worker 和 coordinator 完成后，查看mr-out-*中的输出。完成实验后，输出文件的排序应与顺序输出匹配，如下所示：

```bash
$ cat mr-out-* | sort | more
A 509
ABOUT 2
ACT 8
...
```

我们在main/test-mr.sh中为您提供了一个测试脚本。测试检查wc和indexer MapReduce应用程序在给定pg-xxx.txt时是否产生正确的输出。txt文件作为输入。
测试还检查您的实现是否并行运行Map和Reduce任务，以及您的实现能否从运行任务时崩溃的工作进程中恢复。

如果现在运行测试脚本，它将挂起，因为 coordinator 永远不会完成：

```bash
$ cd ~/6.824/src/main
$ bash test-mr.sh
*** Starting wc test.
```

您可以在mr/coordinator.go中的Done函数中将ret:=false更改为true。让coordinator立即退出。之后

```bash
$ bash test-mr.sh
*** Starting wc test.
sort: No such file or directory
cmp: EOF on mr-wc-all
--- wc output is not the same as mr-correct-wc.txt
--- wc test: FAIL
$
```

测试脚本试图在名为mr-out-X的文件中看到输出，每个reduce任务对应一个。mr/coordinator.go 和 mr/worker.go 的空实现不生成这些数存文件（或做很多其他事情），因此测试失败。

完成后，测试脚本输出应如下所示：

```bash
$ bash test-mr.sh
*** Starting wc test.
--- wc test: PASS
*** Starting indexer test.
--- indexer test: PASS
*** Starting map parallelism test.
--- map parallelism test: PASS
*** Starting reduce parallelism test.
--- reduce parallelism test: PASS
*** Starting crash test.
--- crash test: PASS
*** PASSED ALL TESTS
$
```

您还将看到 Go RPC 包中的一些错误

```bash
2019/12/16 13:27:09 rpc.Register: method "Done" has 1 input parameters; needs exactly three
```

忽略这些消息；将 the coordinator 注册为 RPC 服务器检查其所有方法是否适用于 RPCs（具有3个输入）；这里的Done方法不符合golang RPC规范, 所以会提示警告, 当然这里的Done方法本身也不是给RPC调用的.


### A few rules:

- the map 阶段应该按照 intermediate keys 划分到 nReduce 个桶中,  每个桶对应一个 reduce task, 其中nReduce是reduce的任务数——该参数由 main/mrcoordinator.go 传入到 MakeCoordinator() 函数中。
因此，每个 mapper 都需要生成 nReduce 个 intermediate files,  用于 reduce tasks 进行消费。

- the worker 的实现应该将第X个reduce任务的输出放在文件mr-out-X中。

- mr-out-X文件应包含每个Reduce函数输出的一行记录。该行应使用Golang的“%v %v”格式生成, 这称为"键-值"对。查看main/mrsequential.go 中注释"this is the correct format"。
注意输出格式，否则测试脚本将判定失败。

- 您可以修改mr/worker.go，mr/coordinator.go, 和 mr/rpc.go. 除此外, 您可以临时修改其他文件进行测试，但请确保您的代码与原始版本兼容, 我们将使用原始版本进行测试。

- the worker 应该将 intermediate Map output in files 并放在当前目录中, 稍后将其作为 Reduce Task 的输入进行读取。

- main/mrcoordinator.go 依赖 mr/coordinator.go 去实现一个 Done() 方法, 当MapReduce作业完成时返回true; 此时，mrcoordinator.go 将exit。

- 当作业完全完成时，the worker 进程应退出。简单的实现方法是使用call() 返回值: 如果the worker 未能联系 the coordinator, 它可以假定 
the coordinator 已退出, 因为job is done, 因此the worker 也可以终止。根据您的设计, 您可能会发现, the coordinator 可以向the worker 
提供一个“please exit”的伪任务。

### Hints
- 指南页面提供了一些开发和调试的提示。

- 程序编写如何开始?  一种方法是修改`mr/worker.go的worker()` 来发送RPC给coordinator 来请求一个任务.
然后修改 coordinator 来响应还未被MapTask所处理的文件名。
然后修改 worker 以读取该文件并调用应用程序Map函数, 就像`mrsequential.go`中处理那样。

- 应用程序Map和Reduce函数在运行时使用Go插件包从名称以.so结尾的文件加载。

- 如果您更改`mr/`目录中的任何内容，您可能需要重新构建您使用的任何MapReduce插件, 
比如`go-build-race-buildmode=plugin../mrapps/wc.go`

- 这个实验中 the workers 共享一个文件系统。当所有 worker 都在同一台机器上运行时,这很简单,
但如果工人在不同的机器上运行，则需要像GFS这样的全局文件系统。(global filesystem)

- 中间文件(map task的输出文件)的合理命名约定是`mr-X-Y`,
其中X是 Map Task 编号, Y是 Reduce Task 编号。

- the worker's map task 代码将需要一种方法来将中间 key/value pairs 存储在文件中,
这些数据可以在 reduce task 期间正确读取。一种可能的实现方式是 Go 的 `encoding/json` 包. 
要将 key/value pairs 数据以 JSON 格式 写入到打开的文件中.
```
  enc := json.NewEncoder(file)
  for _, kv := ... {
	  err := enc.Encode(&kv)
```
并且, 读回这些文件数据:
```
  dec := json.NewDecoder(file)
  for {
    var kv KeyValue
    if err := dec.Decode(&kv); err != nil {
      break
    }
    kva = append(kva, kv)
  }
```
- worker的map部分可以使用`ihash(key)`函数(在`worker.go`中)为给定的 key 选择reduce任务。

- 你可以从`mrsequential.go`参考一些代码。用于读取Map输入文件, 用于对Map和Reduce之间的中间 key/value pairs 进行排序, 以及将 Reduce 输出存储在文件中。

- coordinator, 作为RPC服务器, 它是并发的; 不要忘记lock共享数据.

- 使用Go的race detector, `go build -race`和`go run -race test-mr.sh`.

- Workers 有时需要等待, 例如, 直到最后一个 map 任务完成后才能开始 reduce 任务. 
一种可能性是 workers 定期向 coordinator 请求任务, 在每个请求之间通过`time.sleep()`来休眠。另一种可能性是, coordinator 中的相关 RPC Handler 会有一个等待的循环, 比如 `time.Sleep()`或`sync.Cond`。
Go在自己的线程中为每个 RPC 运行 handler, 此时 coordinator 可能更多地处理其他 RPC.

- coordinator 无法可靠地区分 奔溃的 workers, 或者 存活的 workers, 但是由于某种原因而执行太慢或者无法使用。
你能做的最好的事情就是让 coordinator 等待一段时间, 然后放弃并将任务重新分配给另一个 worker. 对于这个实验, 让 coordinator 等待10秒: 超时后, coordinator 应该假设 worker 已经死亡.

- 如果您选择实现 Backup Tasks (第3.6节), 请注意, 我们测试了当 worker 执行任务而不崩溃时, 您的代码不会安排无关任务。
备份任务只能在一段相对较长的时间（例如10秒）后安排。

- 要测试崩溃恢复, 可以使用`mrapps/crash.go`应用插件。它随机出现在Map和Reduce函数中。

- 为了确保没有人在崩溃时观察到部分写入的文件, MapReduce论文提到了使用临时文件并在完全写入后对其进行原子重命名的技巧. 您可以使用`ioutil.TempFile`创建临时文件, 并使用`os.Rename`来原子重命名。

- `test-mr.sh` 运行其子目录`mr-tmp`下的所有进程, 因此, 如果出现问题, 并且您想查看中间文件或输出文件, 请查看这里. 
您可以临时修改`test-mr.sh` 为在测试失败后exit, 进而使得脚本不会继续测试(并覆盖输出文件).

- `test-mr-many.sh` 为运行`test-mr.sh`提供了一个带有超时的基本脚本(这是我们测试你的代码的方式).
它将运行测试的次数作为参数。您不应该运行多个`test-mr.sh`实例来并行, 因为 coordinator 将重用相同的套接字, 从而导致冲突.

- Go RPC 要求远程调用的结构体字段是首字母大写的, 其子结构体也是如此。

- 将响应结构体作为参数传递给 RPC 系统时, *reply 指向的对象应该是 zero-allocated (零值初始化).
RPC调用的代码应始终如下所示:
```
  reply := SomeType{}
  call(..., &reply)
```
见上, 没有被call之前, reply 不能被设置任何字段。
如果不遵循此要求, 当您对字段进行初始化的同时, 也会赋予该字段默认的数据类型, 这是错误的做法, 
当 RPC 将该回复字段设置为默认值时, 可能出现类型错误的问题.

# lab 2

# lab 3

# lab 4
