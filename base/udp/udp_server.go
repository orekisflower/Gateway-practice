package main

import (
	"fmt"
	"net"
)

// UDP 服务器
//
// 步骤：
// 1、监听服务器指定端口
// 2、读取客户端数据
// 3、处理请求并响应

func main() {
	// 1、监听服务器指定端口
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP: net.IPv4(127, 0, 0, 1),
		//端口号，如果不指定，则由操作系统随机分配
		Port: 8080,
	})
	if err != nil {
		fmt.Println("conn failed, err:%v\n", err)
	}

	// 2、读取客户端数据
	//q:以下数组的长度为什么是1024？
	//a:因为UDP协议的数据包大小限制为1024字节
	var data [1024]byte                             //其实byte是uint8的别名
	n, clientAddr, err := conn.ReadFromUDP(data[:]) //n为读取到的字节数，clientaddr为客户端地址
	if err != nil {
		fmt.Println("read error, clientAddr: %v, err: %v\n", clientAddr, err)
	}
	fmt.Printf("clientAddr: %v data: %v count: %v\n", clientAddr, string(data[:n]), n)

	// 3、处理请求并响应
	_, err = conn.WriteToUDP([]byte("received success!"), clientAddr)
	if err != nil {
		fmt.Printf("write error, err: %v\n", err)
	}
}
