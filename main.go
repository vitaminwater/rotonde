package main

import (
	"flag"
	"runtime"

	"github.com/HackerLoop/postman/dispatcher"
	"github.com/HackerLoop/postman/websocketconnection"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	port := flag.Int("port", 4224, "port the websocket will listen on")
	flag.Parse()

	d := dispatcher.NewDispatcher()

	go websocketconnection.Start(d, *port)

	go d.Start()

	select {}
}
