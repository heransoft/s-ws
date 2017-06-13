package s_ws

import (
	"github.com/gorilla/websocket"
	"time"
)

type CloseReason int32

const (
	CloseReason_SelfClose CloseReason = iota
	CloseReason_ReadError
	CloseReason_WriteError
)

type Interface interface {
	//当网络打开时将调用该函数
	OnOpen(*Node)

	//发送数据将异步调用 marshal 函数，调用该函数时需要注意线程安全
	Marshal(int, interface{}) ([]byte, error)

	//接收数据将异步调用 unmarshal 函数，调用该函数时需要注意线程安全
	Unmarshal(int, []byte) (interface{}, error)

	//当接收到数据时将调用该函数
	OnMessage(*Node, interface{}, error)

	//当网络被关闭时将调用该函数
	OnClose(*Node, CloseReason, error)

	//写入数据时如果超时 将关闭网络
	//如果其函数返回值为time.Duration(0)，写入数据将不会进行超时，从而不会因为超时而关闭网络
	GetWriteWait() time.Duration

	//读取数据时如果超时 将关闭网络
	//如果其函数返回值为time.Duration(0)，读取数据将不会进行超时，从而不会因为超时而关闭网络
	GetReadWait() time.Duration
}

const (
	mainChanLength = 1024
)

const (
	readBufferSize  = 64 << 10 // 读消息缓冲区64K
	writeBufferSize = 64 << 10 // 写消息缓冲区64K
)

const (
	open = iota
	writeMessage
	readMessage
	_close
)

type event struct {
	_type int
	arg   interface{}
}

type openEvent struct {
	id   int64
	conn *websocket.Conn
}

type writeMessageEvent struct {
	id int64
	e  error
}

type readMessageEvent struct {
	id int64
	d  interface{}
	ue error
	e  error
}

type closeEvent struct {
	id int64
}
