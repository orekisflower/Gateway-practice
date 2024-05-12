package proxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"
)

type TCPReverseProxy struct {
	//下游真实服务器地址
	Addr string

	DialTimeout     time.Duration //拨号超时
	Deadline        time.Duration //截止时间
	KeepAlivePeriod time.Duration //长连接超时时间

	//TCP拨号方法：net.Dial，这个方法中定义了一个Dialer结构体是拨号的核心对象
	//又返回（调用）Dialer.Dial方法，它封装了Dialer.Dial.DialContext方法
	//这个方法是真正的拨号方法，入参是一个context的空模板、net(网络类型)、add(地址)
	//最后通过系统拨号器sysDialer的dialParallel返回一个net.Conn对象
	DialContext func(ctx context.Context, network, address string) (net.Conn, error)

	//修改响应
	//如果返回错误，将调用ErrorHandler
	ModifyResponse func(*http.Response) error
	//错误处理
	ErrorHandler func(http.ResponseWriter, *http.Request, error)
}

func NewTCPReverseProxy(addr string) *TCPReverseProxy {
	if addr == "" {
		panic("TCP ADDRESS must not be empty!")
	}

	return &TCPReverseProxy{
		Addr:            addr,
		DialTimeout:     10 * time.Second,
		Deadline:        time.Minute,
		KeepAlivePeriod: time.Hour,
	}
}

// ServeTCP TCP服务函数，用于处理TCP连接，实现TCPHandler接口
func (py *TCPReverseProxy) ServeTCP(ctx context.Context, src net.Conn) {
	var cancel context.CancelFunc //检查是否取消操作
	if py.DialTimeout >= 0 {      //连接超时时间
		ctx, cancel = context.WithTimeout(ctx, py.Deadline)
	}
	if py.Deadline >= 0 { //截止时间
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(py.Deadline))
	}
	if cancel != nil {
		defer cancel()
	}

	//完成了取消检查后，开始拨号，拨号参考tcp_client.go中的net.Dial
	if py.DialContext == nil {
		//主要是自定义了结构体参数部分，然后将拨号方法传入拨号器
		//将拨号超时、截止时间、长连接超时时间传入拨号器后，返回该拨号器下执行的拨号方法给DialContext
		py.DialContext = (&net.Dialer{
			Timeout:   py.DialTimeout,
			Deadline:  time.Now().Add(py.Deadline),
			KeepAlive: py.KeepAlivePeriod,
		}).DialContext
	}

	//拨号方法自定义完成后，开始拨号向下游服务器发送请求，返回一个对下游的net.Conn对象
	dst, err := py.DialContext(ctx, "tcp", py.Addr)
	if err != nil {
		// TODO: 错误处理
		return
	}
	defer dst.Close() //记得关闭连接

	//收到请求后，modifyResponse对响应进行修改
	if !py.modifyResponse(dst) {
		//这里的错误通常在modifyResponse中返回
		return
	}

	//将响应拷贝到源连接中，返回给客户端
	//_, err = io.Copy(src, dst)
	//自定义写法
	_, err = bytesCopy(src, dst)
}

// 如果修改成功返回true，否则返回false
func (py *TCPReverseProxy) modifyResponse(res net.Conn) bool {

	return true
}

// 拷贝两个连接的数据
func bytesCopy(src, dst net.Conn) (len int64, err error) {
	_, err = io.Copy(dst, src)
	//q:为什么return中没有len和err也没报错
	//a:因为io.Copy()函数返回的是int64和error，所以这里的返回值也是int64和error
	//直接return裸返回也是对的，但是为了更好的可读性，还是写上了len和err
	return len, err
}
