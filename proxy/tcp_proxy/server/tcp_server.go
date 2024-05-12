package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var (
	//ErrServerClosed 服务已关闭
	ErrServerClosed = errors.New("http: Server closed")
	//ServerContextKey 服务上下文键
	ServerContextKey = &contextKey{"tcp-server"}
	// LocoalAddrContextKey 本地地址的上下文键
	LocoalAddrContextKey = &contextKey{"local-addr"}

	// ErrAbortHandler 错误：中止处理程序
	ErrAbortHandler = errors.New("net/http: abort Handler")
)

// TCPServer TCP代理服务器的核心结构体
// Addr、Handler是TCPServer的两个必选项，handler应提供默认实现
// 以下举例一些常用的配置项
type TCPServer struct {
	Addr    string
	Handler TCPHandler

	// 以下是一些可选项
	BaseContext context.Context //基础上下文,请求的时间、请求的地址、请求的参数信息等
	err         error

	//time.Duration是一个时间段，表示持续的时间长度，默认单位是纳秒
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	KeepAlive    time.Duration

	//互斥锁
	//比如执行连接的关闭、初始化等操作时，需要加锁
	mu sync.Mutex
	//这里用不到太多连接，所以不需要用map
	/*listeners  map[*net.Listener]struct{}
	activeConn map[*conn]struct{}*/

	//doneChan是一个只读的channel，用于通知关闭
	doneChan chan struct{}
	//onShutdown是一个函数切片，用于存储关闭时的回调函数,执行切片内的所有函数后关闭
	//onShutdown []func()
	//以上写法比较复杂，这里用一个简单的int32类型的变量代替，0表示未关闭，1表示关闭
	inShutdown int32
	//onceCloseListener是一个包装过的Listener，，用于控制listener的关闭，防止多次关闭导致panic
	l *onceCloseListener
}

// shuttingDown TCPServer的关闭确认
func (ts *TCPServer) shuttingDown() bool {
	//原子操作，避免并发关闭,1表示已关闭
	return atomic.LoadInt32(&ts.inShutdown) != 0
}

func (ts *TCPServer) ListenAndServe() error {
	//确认服务没有关闭
	if ts.shuttingDown() {
		return ErrServerClosed
	}

	addr := ts.Addr
	//确认地址非空
	if addr == "" {
		return errors.New("we need Address")
	}

	if ts.Handler == nil {
		ts.Handler = &tcpHandler{}
	}

	//Listen方法可以按需创建一个指定类型的Listener，ln就是这个实例
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	//确认服务没有关闭，Address不为空，Listener不为空后，使用ln实例提供服务
	return ts.Serve(ln)
}

// Serve 拿着ListenAndServe中的ln实例提供服务
func (ts *TCPServer) Serve(l net.Listener) error {
	//把ListenAndServe中的ln封装进onceCloseListener，保证只关闭一次
	//同时，onceCloseListener的封装完成，也填充了TCPServer的l属性
	ts.l = &onceCloseListener{Listener: l}
	defer l.Close() //关闭Listener

	//初始化BaseContext
	if ts.BaseContext == nil {
		//BaseContext是一个初始化的上下文，用于存储一些基础信息
		ts.BaseContext = context.Background()
	}

	//获取Ctx
	baseCtx := ts.BaseContext

	//ctx是对baseCtx的一个封装，在baseCtx上下文结构体中增加了ServerContextKey/ts键值对
	ctx := context.WithValue(baseCtx, ServerContextKey, ts)
	for {
		rw, err := l.Accept()
		if err != nil {
			//如果已经结束，返回错误
			if ts.shuttingDown() {
				return ErrServerClosed
			}
			//这里为了简化实现，取消了重试机制
			return err
		}
		//Accept的返回值是一个Conn接口，这里用newConn方法创建一个Conn实例，所以用c接收
		//newConn方法，是对Conn接口的一个封装，增加了一些属性
		//如果没有必要则可以直接使用Conn接口，不用newConn
		c := ts.newConn(rw)
		//http中，这里会对c的rwc，也就是底层链接设置状态，这里是TCP代理，不需要设置状态
		//serve方法是对Conn接口的一个封装，增加了一些方法，生成一个更高级的Conn实例
		go c.serve(ctx)
	}
}

func (ts *TCPServer) newConn(rwc net.Conn) *conn {
	c := &conn{
		server:     ts,                        //当前连接所属的服务器
		rwc:        rwc,                       //底层连接
		remoteAddr: rwc.RemoteAddr().String(), //net.Addr接口原生的String方法
	}

	//从TCPServer中取参数，设置TCPConn的超时参数

	if t := ts.ReadTimeout; t != 0 {
		c.rwc.SetReadDeadline(time.Now().Add(t)) //设置读取超时时间,当前系统时间加上t
	}

	if t := ts.WriteTimeout; t != 0 {
		c.rwc.SetWriteDeadline(time.Now().Add(t)) //设置写超时时间,当前系统时间加上t
	}

	if t := ts.KeepAlive; t != 0 {
		//只有tcp连接才有KeepAlive方法，所以需要断言，将net.Conn接口转换为*net.TCPConn
		if tc, ok := rwc.(*net.TCPConn); ok {
			tc.SetKeepAlive(true)
			tc.SetKeepAlivePeriod(t)
		}
	}

	return c
}

func (c *conn) serve(ctx context.Context) {

	//tcp中不需要响应头，所以这里省略
	//var inFlightResponse *response
	//无论是否有错误，最后都会关闭连接
	defer func() {
		if err := recover(); err != nil && err != ErrAbortHandler {
			//设定一个栈大小，64KB，作为buf的长度
			const size = 64 << 10
			buf := make([]byte, size)
			//runtime.Stack方法，获取当前goroutine的调用栈信息，返回值是int，也就是写入buf的字节数
			//此时的buf是一个切片，所以buf[:n]是一个切片，也就是buf的前n个元素，输出的时候error
			buf = buf[:runtime.Stack(buf, false)]
			fmt.Printf("http: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
		}
		c.rwc.Close()
	}()

	//在上下文中增加本地地址键值对LocoalAddrContextKey/c.rwc.LocalAddr()
	ctx = context.WithValue(ctx, LocoalAddrContextKey, c.rwc.LocalAddr())

	if c.server.Handler == nil {
		panic("http: Server.Handler is nil！")
	}

	//调用TCPHandler接口的ServeTCP方法
	//因为是TCP连接，所以只需要把连接交给客户端处理
	c.server.Handler.ServeTCP(ctx, c.rwc)
}

type conn struct {
	server     *TCPServer //当前连接所属的服务器
	rwc        net.Conn   //当前连接的底层连接
	remoteAddr string     //远程地址，也就是客户端地址
}

type TCPHandler interface {
	//ServeTCP 提供TCP服务
	//ctx是上下文，conn是连接用于读写
	ServeTCP(ctx context.Context, conn net.Conn)
}

type tcpHandler struct{}

func (th *tcpHandler) ServeTCP(ctx context.Context, conn net.Conn) {
	_, err := conn.Write([]byte("Pong! TCP handler here.\n"))
	if err != nil {
		return
	}
}

type contextKey struct {
	name string
}

func ListenAndServe(addr string, handler TCPHandler) error {
	server := &TCPServer{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}

// onceCloseListener wraps a net.Listener, protecting it from
// multiple Close calls.
type onceCloseListener struct {
	//嵌入一个接口，仅是声明这个结构体将会实现接口中声明的所有方法。结构体不会自动继承任何方法，需要提供这些方法的实现
	//onceCloseListener 的实例都必须提供 net.Listener 接口中定义，否则它不会被视为一个完整的 net.Listener
	//嵌入一个类型到结构体中，但没有给这个嵌入的字段一个具体的名字，这种情况该类型自己的名称就被用作字段名。就是所谓的匿名字段或嵌入字段的特性。
	net.Listener
	//结合下面的once.Do方法（once是一个结构体），保证该方法只被执行一次，实现只关闭一次
	once     sync.Once
	closeErr error
}

func (oc *onceCloseListener) Close() error {
	oc.once.Do(oc.close)
	return oc.closeErr
}

func (oc *onceCloseListener) close() { oc.closeErr = oc.Listener.Close() }

// Close 服务器关闭
// 以下处理顺序不能变：
// 需要关闭doneChan，因为它是一个容器，关闭后会释放资源
// inShutdown设置为1，表示关闭
// *onceCloseListener的Close方法会关闭Listener
func (ts *TCPServer) Close() error {
	//为了避免并发关闭，加锁，不能直接ts.inShutdown = 1
	//这里用了原子操作,避免了加锁,类似于事务
	atomic.StoreInt32(&ts.inShutdown, 1)
	close(ts.doneChan) //关闭doneChan
	err := ts.l.Close()
	if err != nil {
		return err
	} //关闭Listener
	return nil
}
