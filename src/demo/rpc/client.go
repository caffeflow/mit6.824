package main

import (
	"fmt"
	"log"
	"net/rpc"
)

type Args struct {
	A, B int
}

type Quotient struct {
	Quo, Rem int
}

func main() {
	c, err := rpc.DialHTTP("tcp", ":1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	// sync call
	args := Args{7, 8}
	var reply int
	err = c.Call("Arith.Multiply", &args, &reply)
	if err != nil {
		log.Fatal("arith error:", err)
	}
	fmt.Printf("Arith: %d*%d=%d\n", args.A, args.B, reply)

	// Async call
	args = Args{7,8} 
	quotient := Quotient{}
	divCall := c.Go("Arith.Divide", &args, &quotient, nil)
	replyCall := <-divCall.Done 
	if replyCall.Error != nil{
		log.Fatal("arith error", replyCall.Error)
	}
	fmt.Printf("Arith: %d/%d=%d...%d\n", args.A, args.B, quotient.Quo, quotient.Rem)
}
