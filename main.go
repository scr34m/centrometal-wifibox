package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eclipse/paho.mqtt.golang/packets"
)

var (
	username   = flag.String("u", "", "web portal username")
	password   = flag.String("p", "", "web portal password")
	localAddr  = flag.String("l", ":1884", "local address")
	remoteAddr = flag.String("r", "portal.centrometal.hr:1883", "remote address")
)

var client mqtt.Client
var proxy *Proxy

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

	var dataDirection string
	if local {
		dataDirection = "C > " + pubPacket.TopicName + " > "
	} else {
		dataDirection = "S < " + pubPacket.TopicName + " < "
	}
	log.Println("Message:", dataDirection+string(pubPacket.Payload))

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

	web := NewWeb(*username, *password)

	ticker := time.NewTicker(60 * time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			if !web.LoggedIn {
				web.Login()
			} else {
				web.Rstat()
			}
		}
	}()

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

	proxy = NewProxy(conn, laddr, raddr, onMessage)
	proxy.Start()
}
