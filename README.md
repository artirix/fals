# Format-Agnostic Log Shipper

The application tails specified logs and POSTs any additions to an API.

## POST payload
The POST request body is a list of one or more objects, with the following specification:

```
[
  {
    "Project":    string,
    "Env":        string,
    "Component":  string,
    "Text":       string,
    "Timestamp":  int64
  }, ...
]
```

## Configuration: config.json
* api_endpoint: HTTP(S) URL to the API endpoint we're going to POST the updates
* api_key: included as a header x-api-key in the POST request
* project: included as Project value in every posted JSON object
* env: included as Env value in every posted JSON object
* shipping_interval: interval to POST requests, in seconds
* components: a list of component names and their respective log files to monitor for updates
  * name: name of the component
  * file: path to log file
