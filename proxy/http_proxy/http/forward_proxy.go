package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	fmt.Println("正向代理服务器启动：8080")
	http.Handle("/", &Pxy{})
	http.ListenAndServe("127.0.0.1:8080", nil)
}

// Pxy 定义一个类型，实现Handler interface
type Pxy struct{}

// ServeHTTP 具体实现方法
func (p *Pxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Printf("Received request %s %s %s: \n", req.Method, req.Host, req.RemoteAddr)

	//1、代理服务器接收客户端请求，赋值，封装成新请求
	//创建了一个新的 http.Request 结构体实例，并将其地址赋值给 outReq 变量，用来接收入参的指针实现浅拷贝
	outReq := &http.Request{} //指针的赋值，需要限制指向变量的类型
	*outReq = *req            //浅拷贝

	//2、发送新请求到下游真实服务器，接收响应
	transport := http.DefaultTransport
	res, err := transport.RoundTrip(outReq)
	if err != nil {
		rw.WriteHeader(http.StatusBadGateway)
		return
	}

	//3、处理响应并返回上游客户端
	//将下游返回的数据，遍历写回发给上游报文的头，拷贝到请求头
	for key, value := range res.Header {
		for _, v := range value {
			//Header()它返回一个 http.Header 类型的值
			//http.Header 实际上是一个 map[string][]string，用于存储 HTTP 响应的头部字段
			rw.Header().Add(key, v)
		}
	}
	//拷贝状态码
	rw.WriteHeader(res.StatusCode)
	//拷贝到请求体
	//rw 是一个 http.ResponseWriter 类型的对象，它实现了 io.Writer 接口，因此可以用于写入响应数据
	//res.Body 是一个实现了 io.ReadCloser 接口的对象，它提供了读取响应体的方法
	//不能将一个读取器（Reader）直接赋值给一个写入器（Writer），而是应该通过io.Copy
	io.Copy(rw, res.Body)
	res.Body.Close()
}
