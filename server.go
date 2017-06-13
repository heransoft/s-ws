package s_ws

import (
	"github.com/gorilla/websocket"
	"net/http"
	"sync/atomic"
	"time"
)

type Server struct {
	allocID  int64
	mainChan chan *event
	listener *websocket.Upgrader
	nodes    map[int64]*Node

	onOpen    func(*Node)
	marshal   func(int, interface{}) ([]byte, error)
	unmarshal func(int, []byte) (interface{}, error)
	onMessage func(*Node, interface{}, error)
	onClose   func(*Node, CloseReason, error)
	writeWait time.Duration
	readWait  time.Duration
}

func NewServerByArguments(
	onOpen func(*Node),
	marshal func(int, interface{}) ([]byte, error),
	unmarshal func(int, []byte) (interface{}, error),
	onMessage func(*Node, interface{}, error),
	onClose func(*Node, CloseReason, error),
	writeWait time.Duration,
	readWait time.Duration,
) *Server {
	p := new(Server)
	p.allocID = 0
	p.mainChan = make(chan *event, mainChanLength)
	p.listener = &websocket.Upgrader{ReadBufferSize: readBufferSize, WriteBufferSize: writeBufferSize}
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

func NewServerByInterface(i Interface) *Server {
	p := new(Server)
	p.allocID = 0
	p.mainChan = make(chan *event, mainChanLength)
	p.listener = &websocket.Upgrader{ReadBufferSize: readBufferSize, WriteBufferSize: writeBufferSize}
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

func (p *Server) GetMainChan() <-chan *event {
	return p.mainChan
}

func (p *Server) Deal(e *event) {
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

func (p *Server) Listen(w http.ResponseWriter, r *http.Request) {
	conn, err := p.listener.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	id := atomic.AddInt64(&p.allocID, 1)
	p.mainChan <- &event{
		_type: open,
		arg: &openEvent{
			id:   id,
			conn: conn,
		},
	}
}
