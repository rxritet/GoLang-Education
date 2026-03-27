// Package hub содержит логику широковещательной рассылки WebSocket-сообщений.
// Изолирует работу с мапой подключений в отдельной горутине через каналы.
package hub

// ClientInterface абстрагирует клиента, чтобы избежать циклической зависимости.
type ClientInterface interface {
	Send() chan []byte
}

// Hub хранит активных клиентов и рассылает сообщения всем.
type Hub struct {
	clients    map[ClientInterface]bool
	broadcast  chan []byte
	Register   chan ClientInterface
	Unregister chan ClientInterface
}

// New создаёт новый хаб.
func New() *Hub {
	return &Hub{
		clients:    make(map[ClientInterface]bool),
		broadcast:  make(chan []byte, 256), // буферизованный канал рассылки
		Register:   make(chan ClientInterface),
		Unregister: make(chan ClientInterface),
	}
}

// Run запускает бесконечный цикл обработки событий хаба.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			// Новый клиент
			h.clients[client] = true

		case client := <-h.Unregister:
			// Отключение клиента
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send())
			}

		case message := <-h.broadcast:
			// Рассылка всем
			for client := range h.clients {
				select {
				case client.Send() <- message:
					// Сообщение успешно поставлено в очередь
				default:
					// Канал клиента переполнен (завис) — отключаем
					close(client.Send())
					delete(h.clients, client)
				}
			}
		}
	}
}

// Broadcast отправляет сообщение в хаб для рассылки всем клиентам.
func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}
