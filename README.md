# Format-Agnostic Log Shipper

The application tails specified logs and ships any additions to an Amazon Kinesis Firehose delivery stream.

## Firehose record payload
The created Firehose record is a list of one or more objects separated with newlines, with the following specification:

```
{"Project": string, "Env": string, "Component": string, "Text": string, "Timestamp": int64}
```

## Configuration: config.json
* aws_region: The region of the Kinesis Firehose delivery stream
* aws_access_key: AWS access key with an attached policy allowing PutRecord on the Firehose stream
* aws_secret_key: AWS secret key
* firehose_stream_name: The Kinesis Firehose delivery stream name
* firehose_record_size: The maximum record size to send to Kinesis Firehose (currently limited to 1024KB by AWS)
* project: included as Project value in every posted JSON object
* env: included as Env value in every posted JSON object
* shipping_interval: interval to POST requests, in seconds
* components: a list of component names and their respective log files to monitor for updates
  * name: name of the component
  * file: path to log file
