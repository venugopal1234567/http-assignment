package httpclient

import (
	"fmt"
	"io"
	"net/http"
)

type HttpClientService interface {
	RetreiveWebsiteData(url string) (string, int, error)
}

type httpClientService struct {
	httpClient *http.Client
}

func NewHttpClientService(c *http.Client) HttpClientService {
	return &httpClientService{
		httpClient: c,
	}
}

func (h *httpClientService) RetreiveWebsiteData(url string) (string, int, error) {
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return "", http.StatusInternalServerError, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode, fmt.Errorf("failed to get data from site")
	}

	b, err := io.ReadAll(resp.Body)
	// b, err := ioutil.ReadAll(resp.Body)  Go.1.15 and earlier
	if err != nil {
		return "", http.StatusInternalServerError, err
	}

	fmt.Println(string(b))
	return string(b), http.StatusOK, nil
}
