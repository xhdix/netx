package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/bassosimone/netx/httpx"
)

// XXX: better handling of HTTP bodies and request IDs
// XXX: better handling of logging

func main() {
	client := httpx.NewClient()
	log.SetLevel(log.DebugLevel)
	client.Dialer.Logger = log.Log
	client.Dialer.EnableTiming = true
	client.Tracer.EventsContainer.Logger = log.Log
	for _, URL := range os.Args[1:] {
		client.Get(URL)
	}
	data, err := json.Marshal(client.HTTPEvents())
	if err != nil {
		log.WithError(err).Fatal("json.Marshal failed")
	}
	fmt.Printf("%s\n", string(data))
	data, err = json.Marshal(client.NetEvents())
	if err != nil {
		log.WithError(err).Fatal("json.Marshal failed")
	}
	fmt.Printf("%s\n", string(data))
}
