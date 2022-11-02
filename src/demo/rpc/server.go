package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

/**
本经验来自  https://cloud.tencent.com/developer/article/1753279

RPC - Remote Procedure Call, 调用远程计算机的程序，就像在本地调用一样。

公司项目一般基于Restful的微服务架构，当微服务之间沟通非常频繁时，就希望用rpc来做内部的通讯，
对外依然用Restful。

golang标准库的rpc包和google的grpc包。这里提及golang中的rpc包。

golang的rpc包支持三个级别：TCP、HTTP、JSONRPC。但是golang中的rpc采用Gob编码，所以只支持
Go开发的服务器和客户端之间的交互,这里我们在内部做通讯，所以可以保证rpc可用。

GOlang的RPC函数的格式如下,才能被远程访问:
func (t *T) MethodName(argType T1, replyType *T2) error
1 函数的首字母大写
2 两个导出类型,第一个是接收的参数,第二个返回客户端的参数,第二个参数是指针类型
3 函数有一个返回值error
4 变量类型必须支持被encoding/gob包编码。

代码思路:
1 在服务端写好调用函数,并注册到rpc服务中.
2 在客户端中远程连接到服务端的rpc服务, 向rpc服务传入调用的方法名和相关参数,等待服务端执行调用函数,再
将结果返回给客户端. 客户端等待返回结果的方式,分为同步和异步.
*/

type Args struct {
	A, B int
}

type Quotient struct {
	Quo, Rem int
}

type Arith int // 对数计算

func (t *Arith) Multiply(args *Args, reply *int) error { // 计算 a * b
	*reply = args.A * args.B
	return nil
}

func (t *Arith) Divide(args *Args, quotient *Quotient) error { // 计算 a / b  和 a % b
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	quotient.Quo = args.A / args.B
	quotient.Rem = args.A % args.B
	return nil
}

func main() {
	arith := new(Arith)
	rpc.Register(arith)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal("listen error:", err)
	}
	http.Serve(l, nil)
}
