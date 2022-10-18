package main

import "time"
import "sync"

func main(){
	var mu sync.Mutex
	counter := 0 
	for i := 0; i < 1000; i++ {
		go func(){
			mu.Lock()
			defer mu.Unlock()
			counter +=1
		}()
	}
	time.Sleep(1 * time.Second)
	println(counter)
}
