package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	mqttbroker "github.com/mochi-co/mqtt/server"
	"github.com/mochi-co/mqtt/server/events"
	"github.com/mochi-co/mqtt/server/listeners"
)

type ClientData struct {
	Username    string
	Password    string
	ClMsgId     int
	SrvMsgId    int
	Token       string
	JsonSignKey []byte
	Topic       string
}

func Server(addr string, key string) {
	server = mqttbroker.New()
	tcp := listeners.NewTCP("t1", addr)

	err := server.AddListener(tcp, nil)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Printf("Local broker listening on %s\n", addr)
		if err := server.Serve(); err != nil {
			log.Fatalf("broker serve error: %v", err)
		}
	}()

	Rstat_Down()

	server.Events.OnConnect = func(cl events.Client, pk events.Packet) {
		log.Printf("OnConnect %s\n", cl.ID)
		clientData = &ClientData{
			Username:    string(pk.Username),
			Password:    string(pk.Password),
			ClMsgId:     0,
			SrvMsgId:    1,
			JsonSignKey: jsonSignInit(key[0:10], key[10:20], key[20:30], key[30:40], string(pk.Password)),
		}
	}

	server.Events.OnSubscribe = func(filter string, cl events.Client, qos byte) {
		log.Printf("OnSubscribe %s: %s\n", cl.ID, filter)
		clientData.Topic = filter
	}

	server.Events.OnDisconnect = func(cl events.Client, err error) {
		log.Printf("OnDisconnect %s: %v\n", cl.ID, err)
	}

	server.Events.OnMessage = func(cl events.Client, pk events.Packet) (pkx events.Packet, err error) {
		pkx = pk
		log.Printf("OnMessage %s: %s\n", cl.ID, string(pk.Payload))

		var m map[string]any
		err = json.Unmarshal(pk.Payload, &m)
		if err != nil {
			panic(err)
		}

		delete(m, "_sign")

		if v, ok := m["_token"].(string); ok {
			clientData.Token = v
			delete(m, "_token")
		}

		if v, ok := m["clMsgId"].(float64); ok {
			clientData.ClMsgId = int(v) + 1
			delete(m, "clMsgId")
		}

		if _, ok := m["_sync"]; ok {
			SyncAck_Down()
			return pk, nil
		}

		if v, ok := m["PING"]; ok {
			Ack_Down("PING", v)
			return pk, nil
		}

		for k := range m {
			if strings.HasPrefix(k, "E") || strings.HasPrefix(k, "W") {
				Ack_Down(k, m[k])

				publishErrorOrWarning(k, m[k])
			}

			publish(k, m[k])
		}

		return pk, nil
	}
}

func Rstat_Down() {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			if clientData == nil {
				continue
			}

			clientData.SrvMsgId++
			msg := []JsonOrderedKV{
				{"RSTAT", "ALL"},
				{"srvMsgId", clientData.SrvMsgId},
			}
			Message_Down(msg)
		}
	}()
}

func SyncAck_Down() {
	if clientData == nil {
		return
	}

	clientData.SrvMsgId++
	msg := []JsonOrderedKV{
		{"_sync_ACK", "ok"},
		{"clMsgId", clientData.ClMsgId},
		{"_token", clientData.Token},
		{"srvMsgId", fmt.Sprintf("%v", clientData.SrvMsgId)},
	}
	Message_Down(msg)
}

func Ack_Down(key string, value any) {
	if clientData == nil {
		return
	}

	clientData.SrvMsgId++
	msg := []JsonOrderedKV{
		{key + "_ACK", value},
		{"srvMsgId", fmt.Sprintf("%v", clientData.SrvMsgId)},
	}
	Message_Down(msg)
}

func Cmd_Down(value any) {
	if clientData == nil {
		return
	}

	clientData.SrvMsgId++
	msg := []JsonOrderedKV{
		{"CMD", value},
		{"srvMsgId", clientData.SrvMsgId},
	}
	Message_Down(msg)
}

func Refresh_Down(value any) {
	if clientData == nil {
		return
	}

	clientData.SrvMsgId++
	msg := []JsonOrderedKV{
		{"REFRESH", value},
		{"srvMsgId", clientData.SrvMsgId},
	}
	Message_Down(msg)
}

func Message_Down(msg []JsonOrderedKV) {
	go func() {
		response := jsonSign(clientData.JsonSignKey, msg)
		server.Publish(clientData.Topic, []byte(response), false)
	}()
}
