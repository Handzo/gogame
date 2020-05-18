package server

import (
	"fmt"
	"io"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type socket struct {
	conn      *websocket.Conn
	writeChan chan []byte
}

// Return socket connection
func newSocket(conn *websocket.Conn) *socket {
	s := &socket{
		conn:      conn,
		writeChan: make(chan []byte, 10),
	}

	go s.writePump()
	return s
}

func (this *socket) close() {
	fmt.Println("CLOSE SOCKET")
	this.conn.Close()
	close(this.writeChan)
}

func (this *socket) writePump() {
	defer func() {
		fmt.Println("stop write pump")
		this.conn.Close()
	}()

	for {
		message, ok := <-this.writeChan
		if !ok {
			this.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		if err := this.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

// Read available message from socket
func (this *socket) ReadMessage() ([]byte, error) {
	_, message, err := this.conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			return nil, err
		}
		return nil, io.EOF
	}

	return message, err
}

// Write message to socket
func (this *socket) WriteMessage(message []byte) {
	this.writeChan <- message
}
