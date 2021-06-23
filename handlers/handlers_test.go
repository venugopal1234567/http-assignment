package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
)

func TestSaveToCache(t *testing.T) {

	tests := []struct {
		name               string
		expectedStatusCode int
		mock               *httpClientServiceMock
	}{
		{
			"happy path",
			http.StatusOK,
			&httpClientServiceMock{
				http.StatusOK,
				"Success",
				nil,
			},
		},
		{
			"happy path",
			http.StatusNotFound,
			&httpClientServiceMock{
				http.StatusNotFound,
				"StatusNotFound",
				nil,
			},
		},
	}
	for _, test := range tests {
		c := cache.New(5*time.Minute, 10*time.Minute)
		ps := ProxyRequestSercive{test.mock, c}
		req, err := http.NewRequest("POST", "/api/assert", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "godoc.org"
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(ps.ProxyServe)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != test.expectedStatusCode {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, test.expectedStatusCode)
		}

	}
}

type httpClientServiceMock struct {
	expectedStatusCode int
	responseBody       string
	expectedError      error
}

func (h *httpClientServiceMock) RetreiveWebsiteData(url string) (string, int, error) {
	return h.responseBody, h.expectedStatusCode, h.expectedError
}
