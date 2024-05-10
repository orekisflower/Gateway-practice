package main

import (
	"fmt"
	"net"
)

// UDP 客户端
//
// 1、连接服务器
// 2、发送数据
// 3、接收数据
func main() {
	// 1、连接服务器，第二位参数为本机地址，第三位参数为服务器地址
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 8080, //服务器监听端口
	})
	if err != nil {
		fmt.Printf("connection failed. err:%v\n", err)
	}

	// 2、发送数据
	data := "Hello UDP Server!"
	_, err = conn.Write([]byte(data)) //把data强转为byte切片类型
	if err != nil {
		fmt.Println("err:", err)
	}

	// 3、接收数据
	result := make([]byte, 1024)
	len, remoteAddr, err := conn.ReadFromUDP(result)
	if err != nil {
		fmt.Println("received failed. err:", err)
	}
	fmt.Printf("response from server, addr:%v data:%v", remoteAddr, string(result[:len]))
}
