package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

// TCP 客户端
// 步骤：
// 1、与服务器建立TCP链接
// 2、接收服务器响应
// 3、向服务器发送消息
// 4、关闭链接，释放资源

func main() {
	//1、与服务器建立TCP链接
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		fmt.Println("connect failed. err:", err)
		return
	}

	//4.关闭链接，释放资源
	defer conn.Close()

	//2、接收服务器响应
	go getServerData(conn)

	//3、向服务器发送消息
	conn.Write([]byte("Hello server!"))
	//os.Stdin代表标准输入流。在大多数情况下指的是键盘输入。
	//NewReader 函数接受一个 io.Reader 接口类型的参数，并返回一个新的 *bufio.Reader 对象。这个新的 Reader 对象会提供比基础 io.Reader 更高效的读取操作，因为它会在内部维护一个缓冲区。
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

// 接收服务器响应
func getServerData(conn net.Conn) {
	//客户端缓冲区
	buf := make([]byte, 1024)
	//循环读取服务器信息并打印
	for {
		n, _ := conn.Read(buf)
		//以下均为健壮性处理
		data := strings.TrimSpace(string(buf[:n])) //TrimSpace用于删除字符串两侧的空白字符（包括空格、制表符、换行符等）
		if data != "" {
			fmt.Println("from Server:", string(buf[:n]))
		}

	}

}
