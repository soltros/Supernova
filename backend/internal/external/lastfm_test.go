package external

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (fn transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }
func TestLastFMHTTP200ErrorIsNotSuccess(t *testing.T) {
	for _, tc := range []struct {
		body string
		fail bool
	}{{`{"error":9,"message":"Invalid session key"}`, true}, {`{"nowplaying":{}}`, false}, {`not json`, true}} {
		c := NewLastFmClient("key", "secret")
		c.client = &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
		})}
		err := c.postAuthenticated(map[string]string{"method": "track.updateNowPlaying"})
		if (err != nil) != tc.fail {
			t.Fatalf("body %s error %v", tc.body, err)
		}
	}
}
