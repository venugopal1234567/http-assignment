package httpclient

import (
	"errors"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
)

type httpMock struct {
	method      string
	statusCode  int
	respnseBody string
}

func benchmarkProxyServe(url string, expectedStatusCode int, mock *httpMock, expectedErr error, b *testing.B) {

	client := &http.Client{}

	p := NewHttpClientService(client)

	for n := 0; n < b.N; n++ {

		httpmock.Activate()

		httpmock.RegisterResponder(mock.method, url,
			httpmock.NewStringResponder(mock.statusCode, mock.respnseBody))

		_, statusCode, err := p.RetreiveWebsiteData(url)
		httpmock.DeactivateAndReset()
		b.Log(statusCode)
		if err != nil {
			if err.Error() != expectedErr.Error() {
				b.Errorf("failed to get data: %v", err)
			}
		}

		if statusCode != expectedStatusCode {
			b.Errorf("Failed to run benchmark : %v", err)
		}

	}
}

func BenchmarkProxyServe(b *testing.B) {
	tests := []struct {
		name               string
		url                string
		httpMock           *httpMock
		expectedStatucCode int
		expectedErr        error
	}{
		{
			"status ok",
			"http://go.org",
			&httpMock{
				"GET",
				http.StatusOK,
				`[{ "name": "Successfully cloned a site"}]`,
			},
			http.StatusOK,
			nil,
		},
		{
			"status not Found",
			"http://go.org",
			&httpMock{
				"GET",
				http.StatusNotFound,
				`Not Found`,
			},
			http.StatusNotFound,
			errors.New("failed to get data from site"),
		},
	}

	for _, test := range tests {
		b.Logf("benchmark name %s", test.name)
		benchmarkProxyServe(test.url, test.expectedStatucCode, test.httpMock, test.expectedErr, b)
	}
}
