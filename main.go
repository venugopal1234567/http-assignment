package main

import (
	"fmt"
	"http-assignment/handlers"
	"http-assignment/internal/domain/httpclient"
	"http-assignment/internal/domain/jwtauth"
	"http-assignment/internal/middlewares"
	"log"
	"net/http"
	"os"
	"time"

	_ "net/http/pprof"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
)

const (
	addr = "127.0.0.1:8001"
)

var rclient *redis.Client

func init() {
	//Initializing redis
	dsn := os.Getenv("REDIS_DSN")
	if len(dsn) == 0 {
		dsn = "localhost:6379"
	}
	rclient = redis.NewClient(&redis.Options{
		Addr: dsn, //redis port
	})
	_, err := rclient.Ping().Result()
	if err != nil {
		panic(err)
	}
}

func main() {
	os.Setenv("ACCESS_SECRET", "jdnfksdmfksd")
	os.Setenv("REFRESH_SECRET", "mcmvmkmsdnfsdmfdsjf")

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	c := cache.New(5*time.Minute, 10*time.Minute)

	client := &http.Client{Transport: tr}
	cs := httpclient.NewHttpClientService(client)
	ja := jwtauth.NewJwtAuthService(os.Getenv("ACCESS_SECRET"), os.Getenv("REFRESH_SECRET"), rclient)
	p := &handlers.ProxyRequestSercive{HttpClient: cs, Cache: c, JwtAuthService: ja}

	m := middlewares.NewMiddleWareService(ja)
	router := mux.NewRouter()

	router.HandleFunc("/api/health", p.HealthCheck)
	proxyServeHandler := http.HandlerFunc(p.ProxyServe)
	router.Handle("/api/assert", m.RequiresLogin(proxyServeHandler))
	router.HandleFunc("/api/login", p.Login)
	router.HandleFunc("/api/logout", p.Logout)
	router.HandleFunc("/api/refresh", p.RefreshToken)

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
