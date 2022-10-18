package main

import (
	"fmt"
	"sync"
)

var num = 0
var mu sync.Mutex
var wg sync.WaitGroup

func main() {
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			sendRPC(id)
			wg.Done()
		}(i)
		
	}
	wg.Wait()
}

func sendRPC(id int) {
	mu.Lock()
	defer mu.Unlock()
	num += 1
	fmt.Printf("id=%d,num=%d\n", id, num)
}
