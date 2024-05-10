package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

func main() {
	var addr = "localhost:8002" //下游真实服务器地址
	http.HandleFunc("/wsHandler", wsHandler)
	log.Println("Starting server at " + addr)
	//如果在启动服务器的过程中发生任何错误，http.ListenAndServe 将返回一个非空的 error
	//如果没有错误发生，http.ListenAndServe 将一直阻塞，直到服务器关闭
	//返回错误的情况下，log.Fatal()会打印错误信息并调用os.Exit(1)退出程序
	log.Fatal(http.ListenAndServe(addr, nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	//相比于http这里创建了一个websocket的升级器
	var upgrader = websocket.Upgrader{} //default options
	//对响应、请求进行升级，返回一个websocket连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer conn.Close()

	go func() {
		for {
			//TextMessage是消息类型，1是纯文本类型，2是BinaryMessage
			err := conn.WriteMessage(1, []byte("心跳检测"))
			if err != nil {
				return
			}
			time.Sleep(3 * time.Second)
		}
	}()

	//以下的read和write是一个死循环，一直在读取和写入
	//conn是一个websocket连接，可以通过它进行读写操作
	//read相当于从客户端读取数据，write相当于向客户端写入数据
	//这里是一个echo server，即客户端发送什么，服务器就返回什么+“lz”
	for {
		mt, msg, err := conn.ReadMessage() //消息类型，消息内容，错误信息
		if err != nil {
			log.Println("read error:", err)
			break
		}
		fmt.Printf("receive mt:%s, msg:%s\n", mt, msg)

		newMsg := string(msg) + "lz"
		msg = []byte(newMsg)
		err = conn.WriteMessage(mt, msg)
		if err != nil {
			log.Println("write error:", err)
			break
		}
	}
}
