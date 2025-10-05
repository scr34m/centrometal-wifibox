package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Web struct {
	LoggedIn bool
	Username string
	Password string
	Token    string
	jar      http.CookieJar
}

func NewWeb(username string, password string) *Web {
	return &Web{
		LoggedIn: false,
		Username: username,
		Password: password,
	}
}

func (w *Web) init() bool {
	log.Println("Web init")

	w.LoggedIn = false
	w.Token = ""
	w.jar, _ = cookiejar.New(nil)

	url := "https://portal.centrometal.hr/login"

	client := &http.Client{Jar: w.jar}

	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	re := regexp.MustCompile(`<input[^>]+name="_csrf_token"[^>]+value="([^"]+)"`)
	matches := re.FindSubmatch(body)
	if len(matches) < 2 {
		log.Println("CSRF does not exists")
		return true
	}
	w.Token = string(matches[1])

	return true
}

func (w *Web) Login() bool {
	w.init()

	time.Sleep(1 * time.Second)

	client := &http.Client{Jar: w.jar}

	postURL := "https://portal.centrometal.hr/login_check"

	data := url.Values{}
	data.Set("_csrf_token", w.Token)
	data.Set("_username", w.Username)
	data.Set("_password", w.Password)

	req, err := http.NewRequest("POST", postURL, strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if strings.Contains(string(body), `id="id-loading-screen-blackout"`) {
		w.LoggedIn = true
		log.Println("Web Login success")
		return true
	} else {
		log.Println("Web Login failed")
		w.LoggedIn = false
		return false
	}
}

func (w *Web) Refresh() bool {
	log.Println("Web Refresh")

	data := []byte(`{"messages":{"5735":{"REFRESH":0}}}`)
	resp := w.post("/api/inst/control/multiple", data)

	if !strings.Contains(resp, `{"status":"success","info":{"permissions":{"5735":2}}}`) {
		log.Println("Web Refresh failed")
		w.LoggedIn = false
		return false
	}

	return true
}

func (w *Web) Rstat() bool {
	log.Println("Web Rstat")

	data := []byte(`{"messages":{"5735":{"RSTAT":"ALL"}}}`)
	resp := w.post("/api/inst/control/multiple", data)

	if !strings.Contains(resp, `{"status":"success","info":{"permissions":{"5735":2}}}`) {
		log.Println("Web Rstat failed")
		w.LoggedIn = false
		return false
	}

	return true
}

func (w *Web) post(uri string, data []byte) string {
	client := &http.Client{
		Jar: w.jar,
	}

	req, err := http.NewRequest("POST", "https://portal.centrometal.hr"+uri, bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return string(body)
}
