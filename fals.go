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
	"github.com/foize/go.fifo"
	"os"
	"time"
)

type Configuration struct {
	Api_endpoint, Api_key, Project, Env        string
	Aws_region, Aws_access_key, Aws_secret_key string
	Firehose_stream_name                       string
	Shipping_interval                          int
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
	fmt.Println("region", configuration.Aws_region)
	fmt.Println(svc)
	streams, err := svc.ListDeliveryStreams(&firehose.ListDeliveryStreamsInput{})
	fmt.Println(streams)
	fmt.Println(err)

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
			for {
				item := outgoing.Next()
				if item == nil {
					break
				}

				error := encoder.Encode(item)

				if error != nil {
					fmt.Println("Encoding error: ", error)
				}

			}
			fmt.Println("Buf", data.String())

			params := &firehose.PutRecordInput{
				DeliveryStreamName: aws.String(configuration.Firehose_stream_name), // Required
				Record: &firehose.Record{ // Required
					Data: data.Bytes(), // Required
				},
			}
			resp, err := svc.PutRecord(params)

			if err != nil {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
				return
			} else {
				fmt.Println(resp)
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
