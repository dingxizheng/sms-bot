package web

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type WebClient struct {
	Number  string
	Channel chan int
}

var upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}
var NumberChannels = map[string]*WebClient{}

func WSHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	nummber := r.URL.Query().Get("nummber")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Could not open websocket connection, error: %v", err)
		http.Error(w, "Could not open websocket connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	closeChan := make(chan struct{})
	numberChan, ok := NumberChannels[id]

	if !ok {
		log.Printf("Could not get existing channel(%v)", id)
		http.Error(w, "Could not get existing channel!", http.StatusBadRequest)
		return
	}

	go func() {
		for {
			mt, _, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Closing websocket connection for number: %v", numberChan.Number)
				close(closeChan)
				delete(NumberChannels, id)
				break
			}

			if mt != websocket.TextMessage {
				log.Printf("Closing websocket connection for number: %v", numberChan.Number)
				close(closeChan)
				delete(NumberChannels, id)
				break
			}
		}
	}()

	for {
		select {
		// on channel close
		case <-closeChan:
			return
		case <-numberChan.Channel:
			if err := conn.WriteJSON(map[string]bool{"refresh": true}); err != nil {
				log.Printf("Failed to send refresh command for number: %v", nummber)
			}
		}
	}
}
