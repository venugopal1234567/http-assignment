package handlers

import (
	"encoding/json"
	"fmt"
	"http-assignment/internal/domain/httpclient"
	"http-assignment/internal/domain/jwtauth"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"
)

type ProxyRequestSercive struct {
	HttpClient     httpclient.HttpClientService
	Cache          *cache.Cache
	JwtAuthService jwtauth.JwtAuthService
}

//A sample use
var user = User{
	ID:       1,
	Username: "username",
	Password: "password",
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

func (p *ProxyRequestSercive) Login(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var u User
	err = json.Unmarshal(body, &u)
	if err != nil {
		log.Printf("Failed to parse request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//compare the user from the request, with the one we defined:
	if user.Username != u.Username || user.Password != u.Password {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode("Please provide valid login details")
		return
	}

	token, err := p.JwtAuthService.CreateToken(user.ID)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(err.Error())
		return
	}
	saveErr := p.JwtAuthService.CreateAuth(user.ID, token)
	if saveErr != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(err.Error())
		return
	}
	tokens := map[string]string{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
	}
	json.NewEncoder(w).Encode(tokens)
}

func (p *ProxyRequestSercive) Logout(w http.ResponseWriter, r *http.Request) {
	tokenString := p.JwtAuthService.ExtractToken(r)
	au, err := p.JwtAuthService.ExtractTokenMetadata(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode("unauthorized")
		return
	}
	deleted, delErr := p.JwtAuthService.DeleteAuth(au.AccessUuid)
	if delErr != nil || deleted == 0 { //if any goes wrong
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode("unauthorized")
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("Successfully logged out")
}

func (p *ProxyRequestSercive) RefreshToken(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	mapToken := map[string]string{}
	err = json.Unmarshal(body, &mapToken)
	if err != nil {
		log.Printf("Failed to parse request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	refreshToken := mapToken["refresh_token"]
	token, err := p.JwtAuthService.VerifyRefreshToken(refreshToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode("unauthorized")
		return
	}

	userId, tokenDetails, err := p.JwtAuthService.RefreshToken(token)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(err.Error())
		return
	}

	saveErr := p.JwtAuthService.CreateAuth(userId, tokenDetails)
	if saveErr != nil {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(saveErr.Error())
		return
	}
	tokens := map[string]string{
		"access_token":  tokenDetails.AccessToken,
		"refresh_token": tokenDetails.RefreshToken,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokens)
}
