package middlewares

import (
	"encoding/json"
	"http-assignment/internal/domain/jwtauth"
	"net/http"
)

type MiddleWareService interface {
	RequiresLogin(h http.Handler) http.Handler
}

type middleWareService struct {
	jwtAuth jwtauth.JwtAuthService
}

func NewMiddleWareService(ja jwtauth.JwtAuthService) MiddleWareService {
	return &middleWareService{
		jwtAuth: ja,
	}
}

func (m *middleWareService) RequiresLogin(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := m.jwtAuth.ExtractToken(r)
		tokenAuth, err := m.jwtAuth.ExtractTokenMetadata(tokenString)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode("Failed to extract token")
			return
		}
		userId, err := m.jwtAuth.FetchAuth(tokenAuth)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(err.Error())
		}
		if tokenAuth.UserId != userId {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode("Unauthorized user")
		}

		h.ServeHTTP(w, r)
	})
}
