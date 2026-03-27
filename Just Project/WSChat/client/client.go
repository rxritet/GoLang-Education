// Package client реализует WebSocket-соединение для каждого пользователя.
package client

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"wschat/hub"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Для разработки разрешаем все Origins
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client — мост между websocket-соединением и Hub.
type Client struct {
	hub  *hub.Hub
	conn *websocket.Conn
	send chan []byte // Буфер исходящих сообщений
	name string
}

// Send (hub.ClientInterface) возвращает канал для отправки сообщений данному клиенту.
func (c *Client) Send() chan []byte {
	return c.send
}

// serveWs обрабатывает WebSocket-upgrade запрос.
func ServeWs(hub *hub.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[client] upgrade error: %v", err)
		return
	}

	name := r.URL.Query().Get("user")
	if name == "" {
		name = "Anonymous"
	}

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
		name: name,
	}
	client.hub.Register <- client

	// Оповещаем чат о новом участнике
	systemMsg, _ := json.Marshal(map[string]string{
		"user": "System",
		"text": client.name + " joined the chat.",
	})
	client.hub.Broadcast(systemMsg)

	// Читаем и пишем в дочерних горутинах (Gorilla WebSocket rule).
	go client.writePump()
	go client.readPump()
}

// readPump читает сообщения из сокета в хаб.
func (c *Client) readPump() {
	defer func() {
		// Оповещаем о выходе
		sysMsg, _ := json.Marshal(map[string]string{
			"user": "System",
			"text": c.name + " left the chat.",
		})
		c.hub.Broadcast(sysMsg)
		c.hub.Unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[client] %s read error: %v", c.name, err)
			}
			break
		}

		// Пакуем сообщение с именем пользователя (в реальном проекте это была бы чёткая структура).
		msgBytes, _ := json.Marshal(map[string]string{
			"user": c.name,
			"text": string(message),
		})
		c.hub.Broadcast(msgBytes)
	}
}

// writePump отгружает сообщения из канала hub в сокет.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			// Поддерживаем соединение живым
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
