package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type JsonOrderedKV struct {
	Key   string `json:"-"`
	Value any    `json:"-"`
}

func jsonSignInit(key1, key2, key3, key4, password string) []byte {
	keys := key1 + key2 + key3 + key4
	return []byte(password + keys[len(password):])
}

func jsonHashGen(key []byte, data string) string {
	h := hmac.New(sha1.New, key)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func jsonValue(v any) any {
	j, _ := json.Marshal(v)
	return j
}

func jsonBuild(msg []JsonOrderedKV) string {
	b := []byte{'{'}
	for i, kv := range msg {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, fmt.Sprintf(`"%s":%s`, kv.Key, jsonValue(kv.Value))...)
	}
	b = append(b, '}')
	return string(b)
}

func jsonSign(key []byte, msg []JsonOrderedKV) string {
	msg = append(msg, JsonOrderedKV{"_sign", jsonHashGen(key, jsonBuild(msg))})
	return jsonBuild(msg)
}
