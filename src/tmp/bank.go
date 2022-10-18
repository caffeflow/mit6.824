package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	a := 1000
	b := 1000
	total := a + b
	var mu sync.Mutex

	// go good_process(&a,&b,mu)
	go bad_process(&a, &b, mu)

	start := time.Now()
	for time.Since(start) < 1*time.Second {
		mu.Lock()
		if a+b != total {
			fmt.Printf("observed violation,a = %v, b=%v, sum=%v\n", a, b, a+b)
		}
		mu.Unlock()
	}

}

func good_process(a, b *int, mu sync.Mutex) { // 传递引用  --- 同时锁住a、b
	for i := 0; i < 100; i++ {
		mu.Lock()
		*a -= 1
		*b += 1
		mu.Unlock()
	}
}

func bad_process(a, b *int, mu sync.Mutex) {
	for i := 0; i < 100; i++ {
		mu.Lock()
		*a -= 1
		mu.Unlock()
		mu.Lock()
		*b += 1
		mu.Unlock()
	}
}
