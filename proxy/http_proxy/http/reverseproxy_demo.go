package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// HTTP反向代理简单版：ReverseProxy实现
func main() {
	//下游真实服务器地址
	realServer := "http://127.0.0.1:8001?a=1&b=2#container"

	//parse解析url成结构体
	serverURL, err := url.Parse(realServer)
	if err != nil {
		log.Println(err)
	}

	//方法内使用解析后的url结构体，创建了一个反向代理对象,算是一个handler
	proxy := httputil.NewSingleHostReverseProxy(serverURL)

	//代理服务器地址
	var addr = "127.0.0.1:8081"

	http.ListenAndServe(addr, proxy)
}
