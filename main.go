package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	mqttbroker "github.com/mochi-co/mqtt/server"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	listenAddr = flag.String("l", ":1883", "local address")
	localAddr  = flag.String("o", "localhost:1883", "local address")
	remoteAddr = flag.String("r", "portal.centrometal.hr:1883", "remote address")
	key        = flag.String("k", "158208e84665e0158208e84665e065e0158208e8", "key used json sign concated: key1, key2, key3, key4")
)

var c chan os.Signal
var client mqtt.Client
var server *mqttbroker.Server

var clientData *ClientData

func publish(k string, v any) {
	v2 := fmt.Sprintf("%v", v)

	topic := fmt.Sprintf("%s/%s", "centrometal", k)
	token := client.Publish(topic, 0, true, v2)
	token.Wait()
	if token.Error() != nil {
		log.Println("Publish error:", token.Error())
	}
}

func command(client mqtt.Client, msg mqtt.Message) {
	v := string(msg.Payload())
	log.Printf("Command call: %s\n", v)
	if v == "CMD ON" {
		Cmd_Down(1)
	} else if v == "CMD OFF" {
		Cmd_Down(0)
	} else if v == "REFRESH" {
		Refresh_Down(0)
	} else if v == "RSTAT" {
		Rstat_Down("ALL")
	} else if strings.HasPrefix(v, "PRD") || strings.HasPrefix(v, "PWR") {
		if strings.Contains(v, ":") {
			parts := strings.SplitN(v, ":", 2)
			Parameter_Down(parts[0], parts[1])
		}
	}
}

func publishErrorOrWarning(k string, v any) {
	v2 := fmt.Sprintf("%v", v)

	if strings.HasPrefix(v2, "P") {
		publish("_e_w_status", "")
	} else {
		publish("_e_w_status", k)
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Printf("Connected to local %s\n", *localAddr)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connect lost: %v", err)
	c <- os.Interrupt
}

func main() {
	flag.Parse()

	log.SetOutput(os.Stdout)

	Server(*listenAddr, *key)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(*localAddr)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client = mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	token := client.Subscribe("centrometal/command", 0, command)
	token.Wait()

	c = make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Println("Shutting down")
}
