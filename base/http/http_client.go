package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// HTTP客户端
// 1、创建客户端
// 2、发起请求
// 3、处理响应
// 4、关闭客户端
func main() {
	// 1、创建客户端
	client := &http.Client{}
	// 2、发起请求
	resp, err := client.Get("http://127.0.0.1:9527/hello")
	// 4、关闭客户端
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}
	// 3、处理响应
	bds, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(bds))
}

//q:以上代码运行后为什么会报错？
