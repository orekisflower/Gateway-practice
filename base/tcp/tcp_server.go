package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// 1、监听服务器指定端口
	//network: 联网方式，必须是tcp，注意小写
	//address: 服务器地址，默认本机
	listener, err := net.Listen("tcp", "127.0.0.1:9090")
	if err != nil {
		fmt.Println("listen failed. err:", err)
		return
	}
	fmt.Println("监听已启动")

	// 2、创建TCP连接
	conn, err := listener.Accept() //链接创建完成之前阻塞
	// 5、释放连接
	defer conn.Close()
	if err != nil {
		fmt.Println("accept failed, err:", err)
	}

	//向客户端发送数据，证明连接成功
	conn.Write([]byte("received success"))

	// 3、处理请求:打印数据到控制台
	//buf作为缓冲区
	go getClientData(conn)

	// 4、对客户端进行响应
	inputReader := bufio.NewReader(os.Stdin)
	for {
		input, _ := inputReader.ReadString('\n')
		input = strings.TrimSpace(input)
		_, err := conn.Write([]byte(input))
		if err != nil {
			fmt.Println("write fialed, err:", err)
			break
		}
	}

}

// 3、处理请求
func getClientData(conn net.Conn) {
	//服务器缓冲区
	buf := make([]byte, 1024)
	//循环读取客户端信息并打印
	for {
		n, _ := conn.Read(buf)
		//以下均为健壮性处理
		data := strings.TrimSpace(string(buf[:n])) //TrimSpace用于删除字符串两侧的空白字符（包括空格、制表符、换行符等）
		if data != "" {
			fmt.Println("from Client:", string(buf[:n]))
		}

	}

}
