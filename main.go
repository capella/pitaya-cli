// Copyright (c) TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"errors"
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya/client"
	ishell "gopkg.in/abiosoft/ishell.v2"
)

var pClient = client.New(logrus.InfoLevel)
var disconnectedCh chan bool

func registerRequest(shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name: "request",
		Help: "makes a request to pitaya server",
		Func: func(c *ishell.Context) {
			if !pClient.Connected {
				c.Err(errors.New("not connected"))
				return
			}
			if len(c.Args) < 1 {
				c.Err(errors.New(`request should be in the format: request {route} [data]`))
				return
			}
			route := c.Args[0]
			var data []byte
			if len(c.RawArgs) > 2 {
				data = []byte(c.RawArgs[2])
			}
			_, err := pClient.SendRequest(route, data)
			if err != nil {
				c.Println(err)
			}
		},
	})
}

func registerNotify(shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name: "notify",
		Help: "makes a notify to pitaya server",
		Func: func(c *ishell.Context) {
			if !pClient.Connected {
				c.Err(errors.New("not connected"))
				return
			}
			if len(c.Args) < 1 {
				c.Err(errors.New(`notify should be in the format: notify {route} [data]`))
				return
			}
			route := c.Args[0]
			var data []byte
			if len(c.RawArgs) > 2 {
				data = []byte(c.RawArgs[2])
			}
			if err := pClient.SendNotify(route, data); err != nil {
				c.Println(err)
				c.Err(err)
			}
		},
	})
}

func registerDisconnect(shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name: "disconnect",
		Help: "disconnects from pitaya server",
		Func: func(c *ishell.Context) {
			if pClient.Connected {
				disconnectedCh <- true
				pClient.Disconnect()
			}
		},
	})
}

func registerConnect(shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name: "connect",
		Help: "connects to pitaya",
		Func: func(c *ishell.Context) {
			if pClient.Connected {
				c.Err(errors.New("already connected"))
				return
			}
			var addr string
			if len(c.Args) == 0 {
				c.Print("address: ")
				addr = c.ReadLine()
			} else {
				addr = c.Args[0]
			}
			err := pClient.ConnectTo(addr)
			if err != nil {
				c.Err(err)
			} else {
				c.Println("connected!")
				disconnectedCh = make(chan bool, 1)
				go readServerMessages(shell)
			}
		},
	})
}

func readServerMessages(c *ishell.Shell) {
	for {
		select {
		case <-disconnectedCh:
			close(disconnectedCh)
			return
		case m := <-pClient.IncomingMsgChan:
			c.Printf("sv-> %s\n", string(m.Data))
		}
	}
}

func configure(c *ishell.Shell) {
	historyPath := os.Getenv("PITAYACLI_HISTORY_PATH")
	if historyPath == "" {
		home, _ := homedir.Dir()
		historyPath = fmt.Sprintf("%s/.pitayacli_history", home)
	}

	c.SetHistoryPath(historyPath)
}

func main() {
	shell := ishell.New()
	configure(shell)

	shell.Println("Pitaya REPL Client")

	registerConnect(shell)
	registerDisconnect(shell)
	registerRequest(shell)
	registerNotify(shell)

	shell.Run()
}
