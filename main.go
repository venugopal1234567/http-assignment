package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
)

const (
	addr = "127.0.0.1:8000"
)

type ProxyRequestSercive struct {
	httpClient *http.Client
	cache      *cache.Cache
}

func (p *ProxyRequestSercive) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// an example API handler
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (p *ProxyRequestSercive) ProxyServe(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if data, isExist := p.cache.Get(r.Host); isExist {
			json.NewEncoder(w).Encode(map[string]string{r.Host: data.(string)})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("Host not found: %s", r.Host)))
		return

	case "POST":
		resp, err := p.httpClient.Get("http://" + r.Host)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("failed to get host: %v", err)))
			return
		}
		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
			w.Write([]byte(fmt.Sprintf("failed to get host: %s", r.Host)))
			return
		}

		b, err := io.ReadAll(resp.Body)
		// b, err := ioutil.ReadAll(resp.Body)  Go.1.15 and earlier
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("failed to read response from : %s", r.Host)))
			return
		}

		fmt.Println(string(b))

		p.cache.Set(r.Host, string(b), 5*time.Minute)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})

	}

}

func main() {

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	c := cache.New(5*time.Minute, 10*time.Minute)

	client := &http.Client{Transport: tr}

	p := &ProxyRequestSercive{httpClient: client, cache: c}
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
