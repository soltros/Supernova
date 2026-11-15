package resourcebudget

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTransportReturnsSaturationInsteadOfUnboundedWaiting(t *testing.T) {
	slots := make(chan struct{}, 1)
	block := make(chan struct{})
	base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/hold" {
			<-block
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})
	tr := &transport{base: base, slots: slots, wait: 20 * time.Millisecond}
	client := &http.Client{Transport: tr}

	firstDone := make(chan error, 1)
	go func() {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.test/hold", nil)
		resp, err := client.Do(req)
		if err == nil {
			err = resp.Body.Close()
		}
		firstDone <- err
	}()

	time.Sleep(5 * time.Millisecond)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.test/next", nil)
	_, err := client.Do(req)
	if !IsSaturated(err) {
		t.Fatalf("expected saturation error, got %v", err)
	}
	close(block)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
}

func TestClosingBodyReleasesSlot(t *testing.T) {
	slots := make(chan struct{}, 1)
	base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})
	client := &http.Client{Transport: &transport{base: base, slots: slots, wait: 20 * time.Millisecond}}

	for i := 0; i < 2; i++ {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.test/", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMetadataResponseBudgetAndSlotRelease(t *testing.T) {
	slots := make(chan struct{}, 1)
	tr := &transport{slots: slots, wait: time.Millisecond, base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		length := int64(-1)
		if r.URL.Path == "/known" {
			length = maxResponseBytes + 1
		}
		return &http.Response{StatusCode: 200, ContentLength: length, Body: io.NopCloser(io.LimitReader(repeatReader{}, maxResponseBytes+1)), Header: make(http.Header), Request: r}, nil
	})}
	client := &http.Client{Transport: tr}
	if _, err := client.Get("https://example.test/known"); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("known length accepted: %v", err)
	}
	resp, err := client.Get("https://example.test/chunked")
	if err != nil {
		t.Fatal(err)
	}
	n, err := io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if n != maxResponseBytes || !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("unbounded read: %d %v", n, err)
	}
	if len(slots) != 0 {
		t.Fatal("slot leaked")
	}
}

type repeatReader struct{}

func (repeatReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }
