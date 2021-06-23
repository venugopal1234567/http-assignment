package handlers

import (
	"encoding/json"
	"fmt"
	"http-assignment/internal/domain/httpclient"
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"
)

type ProxyRequestSercive struct {
	HttpClient httpclient.HttpClientService
	Cache      *cache.Cache
}

func (p *ProxyRequestSercive) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// an example API handler
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (p *ProxyRequestSercive) ProxyServe(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if data, isExist := p.Cache.Get(r.Host); isExist {
			json.NewEncoder(w).Encode(map[string]string{r.Host: data.(string)})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("Host not found: %s", r.Host)))
		return

	case "POST":
		body, statusCode, err := p.HttpClient.RetreiveWebsiteData("http://" + r.Host)
		if err != nil || statusCode != http.StatusOK {
			w.WriteHeader(statusCode)
			w.Write([]byte("failed to get the request"))
			return
		}
		p.Cache.Set(r.Host, body, 5*time.Minute)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}

}
