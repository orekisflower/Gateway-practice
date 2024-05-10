package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	server1 := &RealServer{Addr: "127.0.0.1:8001"}
	server1.Run()

	//接收手动退出的关闭信号，控制服务器的关闭
	//os.Signal的用法，就是生成一个信号的channel，然后用signal.Notify()函数将信号发送到channel中
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

// RealServer 下游真实服务器
type RealServer struct {
	Addr string //服务器主机地址：{host:port}
}

// Run 新建协程启动服务器
// 下面的写法是为了更加了解http的源码运行逻辑
func (r *RealServer) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("/RealServer", r.HelloHandler)
	server := &http.Server{
		Addr:         r.Addr,
		Handler:      mux,
		WriteTimeout: time.Second * 3,
	}
	//这里不需要参数是因为它的入参本来就是用来给http.Server结构体赋值用的，上面已经执行过了
	//以新的协程的方式启动服务
	go func() {
		server.ListenAndServe()
	}()
}

// HelloHandler 路由处理器
func (r *RealServer) HelloHandler(w http.ResponseWriter, req *http.Request) {
	//Sprintf函数用于根据格式化字符串生成一个新的字符串，并返回这个字符串
	newPath := fmt.Sprintf("Here is real server:http://%s%s", r.Addr, req.URL.Path)
	w.Write([]byte(newPath))
	/*
		//这里是一个死循环，一直在写入
		//在http协议中这样操作，客户端是收不到相应的
		//因为http处理是一个请求一个响应，死循环会导致客户端一直等待
		//所以这里是一个错误的写法，就算用goroutine也是不行的
		//因为goroutine是一个协程，它的生命周期是随着主线程的结束而结束的
		for {
			w.Write([]byte(newPath))
			time.Sleep(1 * time.Second)
		}
	*/
}
