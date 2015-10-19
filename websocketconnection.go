package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
)

type Packet struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Start the websocket server, each peer connecting to this websocket will be added as a connection to the dispatcher
func Start(d *Dispatcher, port int) {
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

		startConnection(conn, d)
	})

	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	log.Println(fmt.Sprintf("Websocket server started on port %d", port))
	select {}
}

func startConnection(conn *websocket.Conn, d *Dispatcher) {
	c := NewConnection()
	d.AddConnection(c)
	defer c.Close()

	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		var jsonPacket []byte
		var err error
		var packet Packet

		for {
			select {
			case dispatcherPacket := <-c.InChan:
				switch data := dispatcherPacket.(type) {
				case Event:
					packet = Packet{Type: "event", Payload: data}
				case Action:
					packet = Packet{Type: "action", Payload: data}
				case Definition:
					packet = Packet{Type: "def", Payload: data}
				default:
					log.Info("Oops unknown packet: ", packet)
				}

				jsonPacket, err = json.Marshal(packet)
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

		var dispatcherPacket interface{}

		for {
			messageType, reader, err := conn.NextReader()
			if err != nil {
				log.Println(err)
				errChan <- err
				return
			}
			if messageType == websocket.TextMessage {
				packet := Packet{}
				decoder := json.NewDecoder(reader)
				if err := decoder.Decode(&packet); err != nil {
					log.Warning(err)
					continue
				}

				switch packet.Type {
				case "event":
					event := Event{}
					mapstructure.Decode(packet.Payload, &event)
					dispatcherPacket = event
				case "action":
					action := Action{}
					mapstructure.Decode(packet.Payload, &action)
					dispatcherPacket = action
				case "sub":
					subscription := Subscription{}
					mapstructure.Decode(packet.Payload, &subscription)
					dispatcherPacket = subscription
				case "unsub":
					unsubscription := Unsubscription{}
					mapstructure.Decode(packet.Payload, &unsubscription)
					dispatcherPacket = unsubscription
				case "def":
					definition := Definition{}
					mapstructure.Decode(packet.Payload, &definition)
					dispatcherPacket = definition
				}

				c.OutChan <- dispatcherPacket
			}
		}
	}()

	log.Println("Treating messages")
	wg.Wait()
	log.Println("###########")
}
