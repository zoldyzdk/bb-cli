package api

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDoIncludesAPIErrorFields(t *testing.T) {
	client := NewClient("user", "token")
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{
			"error": {
				"message": "Bad request",
				"fields": {
					"source": ["changeset not found: "],
					"destination": ["changeset not found: "]
				}
			}
		}`
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})

	err := client.Get("/repositories/ws/repo/pullrequests/1", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	msg := err.Error()
	for _, want := range []string{
		"API error (400): Bad request",
		"destination: changeset not found:",
		"source: changeset not found:",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}
