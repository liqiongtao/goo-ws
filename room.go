package goo_ws

type Room struct {
	clients    map[Client]bool
	broadcast  chan []byte
	register   chan Client
	unregister chan Client
}

func NewRoom() *Room {
	return &Room{
		clients:    make(map[Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan Client),
		unregister: make(chan Client),
	}
}

func (r *Room) Run() {
	for {
		select {
		case client := <-r.register:
			r.clients[client] = true
		case client := <-r.unregister:
			if _, ok := r.clients[client]; ok {
				close(client.send)
				delete(r.clients, client)
			}
		case message := <-r.broadcast:
			for client := range r.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(r.clients, client)
				}
			}
		}
	}
}
