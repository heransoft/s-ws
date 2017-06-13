package s_ws

import (
	"github.com/gorilla/websocket"
	"net"
	"sync/atomic"
	"time"
)

type Node struct {
	conn     *websocket.Conn
	id       int64
	mainChan chan *event

	closeFlag int32

	marshal   func(int, interface{}) ([]byte, error)
	unmarshal func(int, []byte) (interface{}, error)
	writeWait time.Duration
	readWait  time.Duration
}

func (n *Node) Close() {
	if n.IsClose() {
		return
	}
	go func() {
		n.mainChan <- &event{
			_type: _close,
			arg: &closeEvent{
				id: n.id,
			},
		}
	}()
}

// 默认使用 binaryMessage 方式进行发送数据
// 如果需要制定 textMessage 方式 可调用SendEx
func (n *Node) Send(data interface{}) {
	n.SendEx(websocket.BinaryMessage, data)
}

func (n *Node) SendEx(messageType int, data interface{}) {
	if n.IsClose() {
		return
	}
	go func() {
		b, e := n.marshal(messageType, data)
		if e != nil {
			return
		}
		n.send(messageType, b)
	}()
}

func (n *Node) GetID() int64 {
	return n.id
}

func (n *Node) LocalAddr() net.Addr {
	return n.conn.LocalAddr()
}

func (n *Node) RemoteAddr() net.Addr {
	return n.conn.RemoteAddr()
}

func (n *Node) IsClose() bool {
	return atomic.LoadInt32(&n.closeFlag) > 0
}

func (n *Node) send(t int, b []byte) {
	if n.writeWait > 0 {
		n.conn.SetWriteDeadline(time.Now().Add(n.writeWait))
	}
	e := n.conn.WriteMessage(t, b)
	n.mainChan <- &event{
		_type: writeMessage,
		arg: &writeMessageEvent{
			id: n.id,
			e:  e,
		},
	}
}

func (n *Node) readMessage() {
	go func() {
		for {
			if n.IsClose() {
				break
			}
			if n.readWait > 0 {
				n.conn.SetReadDeadline(time.Now().Add(n.readWait))
			}
			t, b, e := n.conn.ReadMessage()
			go func() {
				var d interface{}
				var ue error
				if e == nil {
					d, ue = n.unmarshal(t, b)
				}
				n.mainChan <- &event{
					_type: readMessage,
					arg: &readMessageEvent{
						id: n.id,
						d:  d,
						ue: ue,
						e:  e,
					},
				}
			}()
			if e != nil {
				break
			}
		}
	}()
}
