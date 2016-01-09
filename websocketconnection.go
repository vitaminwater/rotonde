package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/HackerLoop/rotonde/shared"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
)

// Start the websocket server, each peer connecting to this websocket will be added as a connection to the dispatcher
func StartWebsocket(d *Dispatcher, port int) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Connection received")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Fatal(err)
		}

		defer conn.Close()

		startWebsocketConnection(conn, d)
	})

	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	log.Println(fmt.Sprintf("Websocket server started on port %d", port))
	select {}
}

func startWebsocketConnection(conn *websocket.Conn, d *Dispatcher) {
	c := NewConnection()
	d.AddConnection(c)
	defer c.Close()

	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case dispatcherPacket := <-c.InChan:
				jsonPacket, err := rotonde.ToJSON(dispatcherPacket)
				if err != nil {
					log.Warning(err)
				}
				if err := conn.WriteMessage(websocket.TextMessage, jsonPacket); err != nil {
					log.Warning(err)
					return
				}
			case <-errChan:
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			messageType, reader, err := conn.NextReader()
			if err != nil {
				log.Println(err)
				errChan <- err
				return
			}
			if messageType == websocket.TextMessage {
				dispatcherPacket, err := rotonde.FromJSON(reader)
				if err != nil {
					log.Warning(err)
				}
				c.OutChan <- dispatcherPacket
			}
		}
	}()

	log.Println("Treating messages")
	wg.Wait()
}
