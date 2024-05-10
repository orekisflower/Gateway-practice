package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var (
	proxyAddr = "127.0.0.1:8082"
	serverURL = "http://127.0.0.1:8002"
)

func main() {
	url, err := url.Parse(serverURL)
	if err != nil {

		log.Println(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(url)
	log.Println("Starting websocket proxy at " + proxyAddr)

	log.Fatal(http.ListenAndServe(proxyAddr, proxy))
}
