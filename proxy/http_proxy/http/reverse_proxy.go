package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

//下游真实服务器 > 代理服务器 > 客户端

//实现步骤：
//0. 启动真实服务器：gateway/proxy/http_proxy/downsteam_real_server.go
//1. 代理接收客户端请求，更改请求结构体信息
//2. 通过负载均衡算法选择真实服务器
//3. 代理服务器向真实服务器转发请求并接收响应
//4. 代理服务器对返回内容做处理并返回给客户端

func main() {
	var port = "8080" //当前代理服务器端口
	http.HandleFunc("/", handler)
	fmt.Println("反向代理服务器启动成功，端口为：" + port)
	http.ListenAndServe(":"+port, nil)
}

var (
	//实际生产环境中，这里应该是一个负载均衡算法，而不是一个固定的地址
	//?a&=1#af是一个无效的查询参数，只是为了演示
	proxyAddr = "http://127.0.0.1:8001?a&=1#af"
)

func handler(w http.ResponseWriter, r *http.Request) {
	//1.解析下游服务器地址，更改请求地址
	//解析代理服务器地址
	//这里返回的是一个指针类型的url.URL结构体，真实返回值是http://127.0.0.1:8001
	realServer, _ := url.Parse(proxyAddr)
	r.URL.Scheme = realServer.Scheme //http
	r.URL.Host = realServer.Host     //127.0.0.1:8001

	//2.请求下游（真实服务器），并获取返回内容
	transport := http.DefaultTransport
	resp, err := transport.RoundTrip(r) //发送请求到真实服务器并接收响应
	defer resp.Body.Close()             //关闭响应体
	if err != nil {
		log.Println(err)
		return
	}

	//3.把下游服务器返回内容返回给客户端
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	bufio.NewReader(resp.Body).WriteTo(w) //将下游服务器返回的内容写入到客户端
}
