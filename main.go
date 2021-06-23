package main

import (
	"fmt"
	"http-assignment/handlers"
	"http-assignment/internal/domain/httpclient"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
)

const (
	addr = "127.0.0.1:8000"
)

func main() {

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	c := cache.New(5*time.Minute, 10*time.Minute)

	client := &http.Client{Transport: tr}
	cs := httpclient.NewHttpClientService(client)
	p := &handlers.ProxyRequestSercive{HttpClient: cs, Cache: c}
	router := mux.NewRouter()

	router.HandleFunc("/api/health", p.HealthCheck)
	router.HandleFunc("/api/assert", p.ProxyServe)

	srv := &http.Server{
		Handler: router,
		Addr:    addr,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	fmt.Printf("listening on: %s", addr)
	log.Fatal(srv.ListenAndServe())
}
