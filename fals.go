package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ActiveState/tail"
	"github.com/foize/go.fifo"
	"net/http"
	"os"
	"time"
)

type Configuration struct {
	Api_endpoint, Api_key, Project, Env string
	Shipping_interval                   int
	Components                          []Component
}

type Component struct {
	Name, File string
}

type Message struct {
	Project, Env, Component, Text string
	Timestamp                     int64
}

func main() {
	// declare our outgoing queue and http client
	outgoing := fifo.NewQueue()
	client := &http.Client{}

	// load the configuration
	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("Error reading config.json:", err)
	}

	// start Filewatchers for the files
	for _, comp := range configuration.Components {
		fmt.Println("Watching", comp.Name, "at", comp.File)
		go Filewatcher(comp, configuration, outgoing)
	}

	// loop
	for {
		outlen := outgoing.Len()
		if outlen > 0 {
			// create a request body
			var data = bytes.NewBuffer([]byte(``))
			encoder := json.NewEncoder(data)

			// add queue contents as payload
			var items []*Message
			for {
				item := outgoing.Next()
				if item == nil {
					break
				}
				items = append(items, item.(*Message))
			}

			error := encoder.Encode(items)
			if error != nil {
				fmt.Println("Encoding error:", error)
				//TODO: Error handling? Abort run?
			}

			// send request
			req, _ := http.NewRequest("POST", configuration.Api_endpoint, data)
			req.Header.Set("x-api-key", configuration.Api_key)

			_, err := client.Do(req)
			if err != nil {
				fmt.Println("error requesting:", err)
			}
		}
		time.Sleep(time.Duration(configuration.Shipping_interval) * time.Second)
	}
}

func Filewatcher(comp Component, conf Configuration, outgoing *fifo.Queue) {
	// tail a given file, add appended lines into a Message queue
	t, _ := tail.TailFile(comp.File, tail.Config{Follow: true})
	for line := range t.Lines {
		outgoing.Add(&Message{
			Project:   conf.Project,
			Env:       conf.Env,
			Component: comp.Name,
			Text:      line.Text,
			Timestamp: time.Now().Unix(),
		})
	}
}
