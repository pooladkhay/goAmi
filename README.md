# goAmi

A minimalistic Go library for dealing with Asterisk Manager Interface.

## Installation

```bash
go get -u github.com/pooladkhay/goAmi
```

## Usage

```go
package main

import (
	"fmt"

	amiClient "github.com/pooladkhay/goAmi"
)

func main() {
	client := &amiClient.Opts{
		Address:  "ASTERISK_ADDR",
		Port:     "AMI_PORT",
		Username: "AMI_USERNAME",
		Secret:   "AMI_PASSWORD",

		PingInterval:      5,
		PongTimeout:       20,
		ReconnectInterval: 2,

		EventsToListen: []string{"All"},
		EventHandler:   eventHandler,
	}

	client.Connect()
	defer client.StartListening()

	if client.Connected {
		// example sending an action
		client.SendAction("Action:PJSIPShowEndpoints")
	}
}

// eventHandler will receive events as map[string]string which later can be filtered accordingly.
func eventHandler(event map[string]string) {
	if event["Event"] != "" {
		fmt.Println(event)
	}
}

```
## Contributing
Pull requests are welcomed. For major changes, please open an issue first to discuss what you would like to change.

## License
[MIT](https://github.com/pooladkhay/goAmi/blob/main/LICENSE)
