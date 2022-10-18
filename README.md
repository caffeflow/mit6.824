[6.824 Schedule: Spring 2022 (mit.edu)](https://pdos.csail.mit.edu/6.824/schedule.html)

# mit6.824


## lab1

a simple sequential mapreduce implementation in `src/main/mrsequential.go` . 

a coupule of mapreduce applications: word-count in `mrapps/wc.go` , and a text indexer in `mrapps/indexer.go` . 

run word-count application as folllows:

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


your job is to implement
