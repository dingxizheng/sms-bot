package web

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebClient struct {
	Number  string
	Channel chan int
}

var upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}
var NumberChannels = map[string]*WebClient{}

func MountWSController(router *gin.Engine) {
	router.GET("/ws", WSHandler)
}

func WSHandler(c *gin.Context) {
	r := c.Request
	w := c.Writer
	id := c.Query("id")
	nummber := c.Query("nummber")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
	}

	defer conn.Close()
	closeChan := make(chan struct{})
	numberChan, ok := NumberChannels[id]

	if !ok {
		return
	}

	go func() {
		for {
			mt, _, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Closing websocket connection for number %v", numberChan.Number)
				close(closeChan)
				delete(NumberChannels, id)
				break
			}

			if mt != websocket.TextMessage {
				log.Printf("Closing websocket connection for number %v", numberChan.Number)
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
				log.Printf("Failed to send refresh command for number %v", nummber)
			}
		}
	}
}
