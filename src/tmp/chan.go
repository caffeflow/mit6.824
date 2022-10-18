package main

func main
(){
	done := make(chan bool)  // 消费生产的阻塞队列 -- thread-safe FIFO 
	for i := 0; i < 5; i++ {
		go func(x int){
			done <- true
			println(x)
		}(i)
	} 
	for i := 0; i < 5; i++ {
		<-done
	}

}

