package s_ws_test

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/heransoft/s-ws"
	"net/http"
	"testing"
	"time"
)

const (
	ws_path      = "/ws"
	port         = ":8088"
	dial_address = "ws://localhost:8088/ws"
)

func Test_Echo(t *testing.T) {
	testServer(1, func(n *s_ws.Node) {
		fmt.Println("server onopen:", n.GetID(), n.LocalAddr(), n.RemoteAddr())
	}, func(n *s_ws.Node, d interface{}, e error) {
		fmt.Println("server onmessage:", n.GetID(), d, e)
		n.SendEx(websocket.TextMessage, []byte("hello"))
	}, func(n *s_ws.Node, r s_ws.CloseReason, e error) {
		fmt.Println("server onclose:", n.GetID(), r, e)
	}, 0, 0, func(index int32, finish chan int64) {
		testClient(1, func(n *s_ws.Node) {
			fmt.Println("client onopen:", n.GetID(), n.LocalAddr(), n.RemoteAddr())
			n.SendEx(websocket.TextMessage, []byte("hello"))
		}, func(n *s_ws.Node, d interface{}, e error) {
			fmt.Println("client onmessage:", d, e)
			n.Close()
			fmt.Println("client close")
		}, func(n *s_ws.Node, r s_ws.CloseReason, e error) {
			fmt.Println("client onclose:", n.GetID(), r, e)
		}, 0, 0, func(index int32, finish chan int64) {
			time.AfterFunc(time.Second*2, func() { finish <- 0 })
		})
		time.AfterFunc(time.Second*2, func() { finish <- 0 })
	})
}

func Test_Echo1(t *testing.T) {
	testServer(1, func(n *s_ws.Node) {
		fmt.Println("server onopen:", n.GetID(), n.LocalAddr(), n.RemoteAddr())
	}, func(n *s_ws.Node, d interface{}, e error) {
		fmt.Println("server onmessage:", n.GetID(), d, e)
		n.SendEx(websocket.TextMessage, []byte("hello"))
	}, func(n *s_ws.Node, r s_ws.CloseReason, e error) {
		fmt.Println("server onclose:", n.GetID(), r, e)
	}, 0, time.Second, func(index int32, finish chan int64) {
		testClient(1, func(n *s_ws.Node) {
			fmt.Println("client onopen:", n.GetID(), n.LocalAddr(), n.RemoteAddr())
			time.AfterFunc(time.Second-time.Millisecond*100, func() {
				n.SendEx(websocket.TextMessage, []byte("hello"))
			})
		}, func(n *s_ws.Node, d interface{}, e error) {
			fmt.Println("client onmessage:", n.GetID(), d, e)
		}, func(n *s_ws.Node, r s_ws.CloseReason, e error) {
			fmt.Println("client onclose:", n.GetID(), r, e)
		}, 0, 0, func(index int32, finish chan int64) {
			time.AfterFunc(time.Second*3, func() { finish <- 0 })
		})
		time.AfterFunc(time.Second*3, func() { finish <- 0 })
	})
}

func testClient(caseCount int32,
	onOpen func(*s_ws.Node),
	onMessage func(*s_ws.Node, interface{}, error),
	onClose func(*s_ws.Node, s_ws.CloseReason, error),
	writeWait time.Duration,
	readWait time.Duration,
	testCase func(int32, chan int64)) {
	client := s_ws.NewClintByArguments(onOpen, func(t int, n interface{}) ([]byte, error) {
		return n.([]byte), nil
	}, func(t int, b []byte) (interface{}, error) {
		return string(b), nil
	}, onMessage, onClose, writeWait, readWait)
	mainThreadExitChan := make(chan int64, 1)
	mainThreadExitedChan := make(chan int64, 1)
	go func() {
		r := int64(0)
		defer func() {
			mainThreadExitedChan <- r
		}()
		for {
			select {
			case result := <-mainThreadExitChan:
				r = result
				return
			case mainChanElement := <-client.GetMainChan():
				client.Deal(mainChanElement)
			}
		}
	}()
	client.Dial(dial_address)
	caseThreadExitedChan := make(chan int64, caseCount)
	for i := int32(0); i < caseCount; i++ {
		index := i
		go func() {
			testCase(index, caseThreadExitedChan)
		}()
	}
	caseThreadExitedCount := int32(0)
	for {
		<-caseThreadExitedChan
		caseThreadExitedCount++
		if caseCount == caseThreadExitedCount {
			mainThreadExitChan <- 0
			<-mainThreadExitedChan
			break
		}
	}
}

func testServer(caseCount int32,
	onOpen func(*s_ws.Node),
	onMessage func(*s_ws.Node, interface{}, error),
	onClose func(*s_ws.Node, s_ws.CloseReason, error),
	writeWait time.Duration,
	readWait time.Duration,
	testCase func(int32, chan int64)) {
	server := s_ws.NewServerByArguments(onOpen, func(t int, n interface{}) ([]byte, error) {
		return n.([]byte), nil
	}, func(t int, b []byte) (interface{}, error) {
		return string(b), nil
	}, onMessage, onClose, writeWait, readWait)
	mainThreadExitChan := make(chan int64, 1)
	mainThreadExitedChan := make(chan int64, 1)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc(ws_path, server.Listen)
		http.ListenAndServe(port, mux)
	}()
	go func() {
		r := int64(0)
		defer func() {
			mainThreadExitedChan <- r
		}()
		for {
			select {
			case result := <-mainThreadExitChan:
				r = result
				return
			case mainChanElement := <-server.GetMainChan():
				server.Deal(mainChanElement)
			}
		}
	}()
	caseThreadExitedChan := make(chan int64, caseCount)
	for i := int32(0); i < caseCount; i++ {
		index := i
		go func() {
			testCase(index, caseThreadExitedChan)
		}()
	}
	caseThreadExitedCount := int32(0)
	for {
		<-caseThreadExitedChan
		caseThreadExitedCount++
		if caseCount == caseThreadExitedCount {
			mainThreadExitChan <- 0
			<-mainThreadExitedChan
			break
		}
	}
}
