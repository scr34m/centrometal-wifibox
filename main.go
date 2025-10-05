// mosquitto_sub -v -h portal.centrometal.hr -p 1883 -t 'cm/inst/cmpelet/A62EC70C' -P 41E8C1F7 -u A62EC70C

package main

import (
	"flag"
	"log"
	"net"
	"os"
	"bytes"
	"fmt"
	"encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eclipse/paho.mqtt.golang/packets"
)

var (
	localAddr    = flag.String("l", ":1884", "local address")
	remoteAddr   = flag.String("r", "portal.centrometal.hr:1883", "remote address")
)

var client mqtt.Client

func onMessage(local bool, b []byte) {
    reader := bytes.NewReader(b)
    packet, err := packets.ReadPacket(reader)
    if err != nil {
        log.Println("Error reading packet:", err)
        return
    }

    pubPacket, ok := packet.(*packets.PublishPacket)
    if !ok {
        return
    }

    var payload map[string]interface{}
    if err := json.Unmarshal(pubPacket.Payload, &payload); err != nil {
        log.Println("JSON parse error:", err)
        return
    }

    for k, v := range payload {
        topic := fmt.Sprintf("%s/%s", "centrometal", k)
        msg := fmt.Sprintf("%v", v)
        token := client.Publish(topic, 0, true, msg)
        token.Wait()
        if token.Error() != nil {
            log.Println("Publish error:", token.Error())
        } else {
            log.Printf("Sent %s -> %s\n", topic, msg)
        }
    }
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
    log.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
    log.Printf("Connect lost: %v", err)
}

func main() {
	flag.Parse()

    log.SetOutput(os.Stdout)

	log.Printf("Proxing from %v to %v\n", *localAddr, *remoteAddr)

	laddr, err := net.ResolveTCPAddr("tcp", *localAddr)
	if err != nil {
		log.Printf("Failed to resolve local address: %s", err)
		os.Exit(1)
	}

	raddr, err := net.ResolveTCPAddr("tcp", *remoteAddr)
	if err != nil {
		log.Printf("Failed to resolve remote address: %s", err)
		os.Exit(1)
	}

	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		log.Printf("Failed to open local port to listen: %s", err)
		os.Exit(1)
	}

	conn, err := listener.AcceptTCP()
	if err != nil {
		log.Printf("Failed to accept connection '%s'", err)
		os.Exit(1)
	}

    opts := mqtt.NewClientOptions()
    opts.AddBroker("localhost:1883")
    opts.OnConnect = connectHandler
    opts.OnConnectionLost = connectLostHandler
    client = mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        panic(token.Error())
    }

	proxy := NewProxy(conn, laddr, raddr, onMessage)
	proxy.Start()
}
