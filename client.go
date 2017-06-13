package s_ws

import (
	"github.com/gorilla/websocket"
	"sync/atomic"
	"time"
)

type Client struct {
	allocID  int64
	mainChan chan *event
	nodes    map[int64]*Node

	onOpen    func(*Node)
	marshal   func(int, interface{}) ([]byte, error)
	unmarshal func(int, []byte) (interface{}, error)
	onMessage func(*Node, interface{}, error)
	onClose   func(*Node, CloseReason, error)
	writeWait time.Duration
	readWait  time.Duration
}

func NewClintByArguments(
	onOpen func(*Node),
	marshal func(int, interface{}) ([]byte, error),
	unmarshal func(int, []byte) (interface{}, error),
	onMessage func(*Node, interface{}, error),
	onClose func(*Node, CloseReason, error),
	writeWait time.Duration,
	readWait time.Duration,
) *Client {
	p := new(Client)
	p.allocID = 0
	p.mainChan = make(chan *event, mainChanLength)
	p.nodes = make(map[int64]*Node)

	p.onOpen = onOpen
	p.marshal = marshal
	p.unmarshal = unmarshal
	p.onMessage = onMessage
	p.onClose = onClose
	p.writeWait = writeWait
	p.readWait = readWait
	return p
}

func NewClientByInterface(i Interface) *Client {
	p := new(Client)
	p.allocID = 0
	p.mainChan = make(chan *event, mainChanLength)
	p.nodes = make(map[int64]*Node)

	p.onOpen = i.OnOpen
	p.marshal = i.Marshal
	p.unmarshal = i.Unmarshal
	p.onMessage = i.OnMessage
	p.onClose = i.OnClose
	p.writeWait = i.GetWriteWait()
	p.readWait = i.GetReadWait()
	return p
}

func (p *Client) GetMainChan() <-chan *event {
	return p.mainChan
}

func (p *Client) Deal(e *event) {
	switch e._type {
	case open:
		arg := e.arg.(*openEvent)
		node := &Node{
			id:        arg.id,
			conn:      arg.conn,
			mainChan:  p.mainChan,
			closeFlag: 0,
			marshal:   p.marshal,
			unmarshal: p.unmarshal,
			writeWait: p.writeWait,
			readWait:  p.readWait,
		}
		p.nodes[arg.id] = node
		p.onOpen(node)
		node.readMessage()
	case readMessage:
		arg := e.arg.(*readMessageEvent)
		node, exist := p.nodes[arg.id]
		if exist {
			if arg.e != nil {
				atomic.AddInt32(&node.closeFlag, 1)
				node.conn.Close()
				p.onClose(node, CloseReason_ReadError, arg.e)
				delete(p.nodes, arg.id)
			} else {
				p.onMessage(node, arg.d, arg.ue)
			}
		}
	case writeMessage:
		arg := e.arg.(*writeMessageEvent)
		node, exist := p.nodes[arg.id]
		if exist {
			if arg.e != nil {
				atomic.AddInt32(&node.closeFlag, 1)
				node.conn.Close()
				p.onClose(node, CloseReason_WriteError, arg.e)
				delete(p.nodes, arg.id)
			}
		}
	case _close:
		arg := e.arg.(*closeEvent)
		node, exist := p.nodes[arg.id]
		if exist {
			atomic.AddInt32(&node.closeFlag, 1)
			node.conn.Close()
			p.onClose(node, CloseReason_SelfClose, nil)
			delete(p.nodes, arg.id)
		}
	}
}

func (p *Client) Dial(url string) {
	go func() {
		conn, _, e := websocket.DefaultDialer.Dial(url, nil)
		if e != nil {
			return
		}
		id := atomic.AddInt64(&p.allocID, -1)
		p.mainChan <- &event{
			_type: open,
			arg: &openEvent{
				id:   id,
				conn: conn,
			},
		}
	}()
}
