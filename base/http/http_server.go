package main

import "net/http"

// 1、注册路由
//
//	设置路由规则，即访问路径
//	定义该路由规则的处理器：回调函数
//
// 2、启动服务
func main() {
	// 注册路由
	//其中w是写入到客户端的数据也就是响应，r是从客户端读取的数据也就是请求
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello, this is server!"))
	})

	// 启动服务
	http.ListenAndServe("127.0.0.1:9527", nil)
}
