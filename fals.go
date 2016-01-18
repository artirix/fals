package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ActiveState/tail"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/mjuuso/go.fifo"
	"os"
	"time"
)

type Configuration struct {
	Api_endpoint, Api_key, Project, Env        string
	Aws_region, Aws_access_key, Aws_secret_key string
	Firehose_stream_name                       string
	Shipping_interval, Firehose_record_size    int
	Components                                 []Component
}

type Component struct {
	Name, File string
}

type Message struct {
	Project, Env, Component, Text string
	Timestamp                     int64
}

func main() {
	// load the configuration
	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("Error reading config.json:", err)
	}

	// declare our outgoing queue
	outgoing := fifo.NewQueue()

	// initialize aws sdk and credentials from configuration
	svc := firehose.New(session.New(), &aws.Config{Region: aws.String(configuration.Aws_region),
		Credentials: credentials.NewStaticCredentials(configuration.Aws_access_key, configuration.Aws_secret_key, "")})

	_, err = svc.ListDeliveryStreams(&firehose.ListDeliveryStreamsInput{})
	if err != nil {
		fmt.Println("Error listing delivery streams:", err)
		return
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

			// add queue contents as payload, not exceeding Firehose_record_size
			for {
				// get the next item in the queue
				item := outgoing.Peek()
				if item == nil {
					// no more items in queue
					break
				}

				json_value, err := json.Marshal(item)
				if err != nil {
					fmt.Println("Encoding error:", err)
					break
				}

				if (bytes.NewBuffer(json_value).Len() + data.Len()) > configuration.Firehose_record_size {
					// record size would be exceeded
					break
				}

				// encode the item to payload buffer, while removing it from the queue
				error := encoder.Encode(outgoing.Next())

				if error != nil {
					fmt.Println("Encoding error:", error)
				}

			}

			// create the Firehose record
			params := &firehose.PutRecordInput{
				DeliveryStreamName: aws.String(configuration.Firehose_stream_name),
				Record: &firehose.Record{
					Data: data.Bytes(),
				},
			}

			// send the record to Firehose
			resp, err := svc.PutRecord(params)

			if err != nil {
				fmt.Println(err.Error())
				return
			} else {
				fmt.Println(resp)
			}
		}
		// sleep Shipping_interval before the next iteration
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
