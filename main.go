package main

import (
	"flag"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	port := flag.Int("port", 4224, "port the websocket will listen on")
	flag.Parse()

	d := NewDispatcher()

	go Start(d, *port)

	go d.Start()

	select {}
}
