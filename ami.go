package goAmi

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

type Opts struct {
	Address   string
	Port      string
	Username  string
	Secret    string
	Connected bool

	// Interval to send PING action to keep connection alive, in seconds.
	PingInterval time.Duration

	// Time to reconnect after no ping response received.
	// It has to be grater than PingInterval
	PongTimeout time.Duration

	// Interval to try reconnecting when an error occured, in seconds.
	ReconnectInterval time.Duration

	// Takes single or multiple events as a slice of strings and a handler function which receives events.
	// Pass []string{"All"} to listen for all events.
	EventsToListen []string

	// a func in which events will be received.
	EventHandler func(map[string]string)

	conn           *net.TCPConn
	eventChan      *chan map[string]string
	pingerDoneChan *chan bool
}

func (o *Opts) Connect() {
	if _networkAvailable() {
		srvAddr := o.Address + ":" + o.Port

		eCh := make(chan map[string]string)
		o.eventChan = &eCh

		piCh := make(chan bool)
		o.pingerDoneChan = &piCh

		tcpAddr, err := net.ResolveTCPAddr("tcp", srvAddr)
		o._checkError(err, true)

		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		o._checkError(err, true)

		o.conn = conn

		o.Connected = true

		loginAction := fmt.Sprintf("Action:Login\r\nUsername:%v\r\nSecret:%v", o.Username, o.Secret)

		_, err = conn.Write([]byte(loginAction + "\r\n\r\n"))
		o._checkError(err, false)

		o._pinger()
		o._eventParser()
	} else {
		o._checkError(errors.New("network unavailable"), true)
	}
}

// You need to defer it since is has to be the last line to get executed.
func (o *Opts) StartListening() {
	if _networkAvailable() {
		for {
			e := <-*o.eventChan
			for _, event := range o.EventsToListen {
				if event != "All" && event != "all" {
					if strings.Compare(e["Event"], event) == 0 {
						// e represents filtered events
						o.EventHandler(e)
					}
				} else {
					// e represents all events
					o.EventHandler(e)
				}
			}
		}
	}
}

// Sends a new action to Asterisk AMI.
// Action must be a string with the following format: "Action: ActionName".
// e.g. "Action: PING"
func (o *Opts) SendAction(action string) {
	_, err := o.conn.Write([]byte(action + "\r\n\r\n"))
	o._checkError(err, false)
}

func (o *Opts) _eventParser() {
	go func() {
		for {
			o.conn.SetReadDeadline(time.Now().Add(o.PongTimeout * time.Second))

			resp := make([]byte, 2048)
			_, err := o.conn.Read(resp)
			o._checkError(err, true)

			var mf []byte

			for _, b := range resp {
				if b != 0 {
					if b == 10 || b == 13 {
						mf = append(mf, 35)
					} else {
						mf = append(mf, b)
					}
				}
			}

			sp := strings.Split(string(mf), "####")

			for _, v := range sp {
				mp := make(map[string]string)
				if len(v) != 0 {
					event := strings.Split(v, "##")
					for _, i := range event {
						if strings.Contains(i, ": ") {
							si := strings.Split(i, ": ")
							if len(si) == 2 {
								if si[0] != "" && si[1] != "" {
									mp[si[0]] = si[1]
								}
							}
						} else {
							if len(i) != 0 {
								fmt.Println(i + "\r\n")
							}
						}
					}
				}
				if len(mp) != 0 {
					*o.eventChan <- mp
				}
			}
		}
	}()
}

func (o *Opts) _pinger() {
	ticker := time.NewTicker(o.PingInterval * time.Second)

	go func() {
		for {
			select {
			case <-*o.pingerDoneChan:
				fmt.Println("Pinger Terminated.")
				ticker.Stop()
				close(*o.pingerDoneChan)
				return
			case <-ticker.C:
				fmt.Println("Ping...")
				_, err := o.conn.Write([]byte("Action: PING" + "\r\n\r\n"))
				o._checkError(err, false)
			}
		}
	}()
}

func (o *Opts) _reboot() {
	o.Connect()
	o.StartListening()
}

func (o *Opts) _checkError(err error, reboot bool) {
	if err != nil {
		fmt.Println("_checkError:", err)
		if reboot {
			o.Connected = false
			for {
				if o.conn != nil {
					*o.pingerDoneChan <- true
					o.conn.Close()
					o.conn = nil
				}
				if _networkAvailable() {
					o._reboot()
					return
				}
				fmt.Println("Network Unavailable, Reconnecting...")
				time.Sleep(o.ReconnectInterval * time.Second)
			}
		}
	}
}

func _networkAvailable() bool {
	_, err := net.ResolveTCPAddr("tcp", "google.com:443")
	return err == nil
}
