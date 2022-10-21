[6.824 Schedule: Spring 2022 (mit.edu)](https://pdos.csail.mit.edu/6.824/schedule.html)

# basic

- go 1.19.2
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

### 介绍

在这个实验室中，你将建立一个MapReduce系统。你将实现一个调用应用程序Map和Reduce函数并处理读写文件的worker进程，以及一个向worker分派任务并处理失败worker的协调者进程。你将构建类似于MapReduce论文的东西。(注意：本实验室使用 "协调器 "而不是论文中的 "主控器"）。

### kais

# lab 2

# lab 3

# lab 4
