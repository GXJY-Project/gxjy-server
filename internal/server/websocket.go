package server

import (
	"errors"
	"net/http"

	"github.com/gorilla/websocket"
)

const maxWebSocketBody = 16 << 20

var errWebSocketFrame = errors.New("WebSocket 只接受二进制消息")

var gameUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(_ *http.Request) bool {
		// 离线测试服务允许所有来源连接。
		return true
	},
}

type webSocketConn struct {
	conn *websocket.Conn
}

func upgradeWebSocket(writer http.ResponseWriter, request *http.Request) (*webSocketConn, error) {
	connection, err := gameUpgrader.Upgrade(writer, request, nil)
	if err != nil {
		return nil, err
	}
	connection.SetReadLimit(maxWebSocketBody)
	return &webSocketConn{conn: connection}, nil
}

func (connection *webSocketConn) readMessage() ([]byte, error) {
	messageType, body, err := connection.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if messageType != websocket.BinaryMessage {
		return nil, errWebSocketFrame
	}
	return body, nil
}

func (connection *webSocketConn) writeBinary(body []byte) error {
	return connection.conn.WriteMessage(websocket.BinaryMessage, body)
}

func (connection *webSocketConn) close() error {
	return connection.conn.Close()
}
