package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

//HTTP反向代理完整版：用ReverseProxy结构体实现

//支持功能：URL重写、更改内容、错误信息回调、连接池

func main() {
	//下游真实服务器地址
	realServer := "http://127.0.0.1:8001?a=1&b=2#container"

	//parse解析url成结构体
	serverURL, err := url.Parse(realServer)
	if err != nil {
		log.Println(err)
	}

	//方法内使用解析后的url结构体，创建了一个反向代理对象,算是一个handler
	proxy := NewSingleHostReverseProxy(serverURL)

	//代理服务器地址
	var addr = "127.0.0.1:8081"

	log.Println("Starting proxy http server at:" + addr)
	http.ListenAndServe(addr, proxy)
}

// 在包外通过http.调用Transport结构体的方法，实现连接池的自定义
var transport = &http.Transport{
	//之所以不设置Proxy，是因为这段代码本身就是代理
	//Proxy: ProxyFromEnvironment,

	//defaultTransportDialContext 在这里是一个函数，该函数接收一个 *net.Dialer 作为参数
	//并返回一个与 net.Dialer 的 DialContext 方法签名相同的函数或接口类型
	//这种写法的目的是为了封装或扩展 DialContext 方法的默认行为
	//DialContext: defaultTransportDialContext

	//以下写法是为了更加了解http的源码运行逻辑，所以将默认的DialContext方法展开，自定义了一些参数
	//1、创建了具体的net.Dialer指针
	//2、并访问了其DialContext方法，使其作为了外部DialContext参数的值
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	//默认就是true
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second, //TLS握手超时时间
	ExpectContinueTimeout: 1 * time.Second,  //100code响应超时时间，通常用于发送大文件时出现
}

func NewSingleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		rewriteRequestURL(req, target)
	}

	//修改返回响应体
	modifyResponse := func(res *http.Response) error {
		fmt.Println("modifyResponse Func")

		//如果状态码是101，表示协议切换，不需要修改
		if res.StatusCode == 101 {
			//strings.Contains 函数检查从 HTTP 响应头中获取的 "Connection" 字段值是否包含子串 "Upgrade"
			if strings.Contains(res.Header.Get("Connection"), "Upgrade") {
				return nil
			}
		}

		if res.StatusCode == 200 {
			//获取响应体
			srcBody, err := ioutil.ReadAll(res.Body)
			if err != nil {
				panic(err)
			}

			//拼接一个新的字符串，加入字节切片，构成新的响应体
			newBody := []byte(string(srcBody) + "lz")
			//接收字节类型数组，转换为一个新的Buffer，Buffer就是一个Reader和Writer,这里的方法只取了Reader部分
			res.Body = ioutil.NopCloser(bytes.NewBuffer(newBody))
			//响应头中的文本长度字段，需要重新设置，并不会自动更新
			res.ContentLength = int64(len(newBody))

			//修改响应头
			length := int64(len(newBody))
			//string转int，10代表十进制
			res.Header.Set("Content-Length", strconv.FormatInt(length, 10))
		}
		//return nil，通常如此，以下是为了测试错误回调函数
		return errors.New("出错了")
	}

	//错误回调函数，后台出现错误时会调用这个函数
	//为空时，返回502 Bad Gateway
	errFunc := func(w http.ResponseWriter, r *http.Request, err error) {
		fmt.Println("errorFunc")
		http.Error(w, "ErrorHandler error"+err.Error(), http.StatusInternalServerError)
	}

	//返回设定好的反向代理实例，其中赋值了自定义的代理函数
	return &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: modifyResponse,
		ErrorHandler:   errFunc,
		Transport:      transport,
	}
}

func rewriteRequestURL(req *http.Request, target *url.URL) {
	targetQuery := target.RawQuery
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	//RawPath就是请求参数，Path是请求路径
	//原代码：req.URL.Path, req.URL.RawPath = joinURLPath(target, req.URL)
	//之所以要这样改，是因为 Go 的 net/http 包会自动处理 URL 的解码和编码通常遇不到RawPath的情况
	req.URL.Path = joinURLPath(target.Path, req.URL.Path)
	if targetQuery == "" || req.URL.RawQuery == "" {
		req.URL.RawQuery = targetQuery + req.URL.RawQuery
	} else {
		req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
	}
}

// joinURLPath函数用于拼接两个路径，a在前，且不能有多余的"/"
// a: "" or "/"
// b: /realServer ""
func joinURLPath(a, b string) string {
	//防止有多余的"/"，去掉a的最后一个为/的字符
	aSlash := strings.HasSuffix(a, "/")
	bSlash := strings.HasPrefix(b, "/")
	switch {
	case aSlash && bSlash:
		//截掉b的第一个字符
		return a + b[1:]
	case aSlash || bSlash:
		return a + b
	}
	return a + "/" + b
}
