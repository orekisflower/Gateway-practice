package main

import (
	"context"
	"fmt"
	"gateway/proxy/tcp_proxy/proxy"
	"gateway/proxy/tcp_proxy/server"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

//TCP代理服务器，实现服务与代理分离

// 参考http服务的流程搞了一个TCP服务器
func main() {
	var (
		tcpServerAddr = "127.0.0.1:8003"
		tcpProxyAddr  = "127.0.0.1:8083"
	)

	//启动TCP代理服务器
	go func() {

		//1、创建TCPServer实例
		tcpServer := &server.TCPServer{
			Addr: tcpServerAddr,
			//以下&tcpHandler{}是实现了TCPHandler接口的对象
			//与http不同的是，这里是TCP，直接把连接交给客户端处理，实现代理连接读写
			Handler: &Handler{},
		}

		//2、启动监听提供服务
		log.Println("Starting TCP Server at " + tcpServerAddr)

		//本质上是做了&Server{}结构体初始化，和它的ListenAndServe()方法的封装
		err := tcpServer.ListenAndServe()
		if err != nil {
			return
		}
	}()

	//启动TCP代理客户端
	go func() {
		//1、创建TCP代理实例
		tcpProxy := proxy.NewTCPReverseProxy(tcpServerAddr)
		//2、启动监听提供服务
		fmt.Println("Starting TCP Proxy at " + tcpProxyAddr)
		err := server.ListenAndServe(tcpProxyAddr, tcpProxy)
		if err != nil {
			return
		}
	}()

	quit := make(chan os.Signal)
	//Signal Interrupt和Signal Terminate
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
}

// Handler 负责具体实现TCPHandler接口的对象
type Handler struct {
}

func (th *Handler) ServeTCP(ctx context.Context, conn net.Conn) {
	_, err := conn.Write([]byte("haha.\n"))
	if err != nil {
		return
	}
}
