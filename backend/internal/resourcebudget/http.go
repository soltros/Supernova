package resourcebudget

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

var ErrSaturated = errors.New("outbound request budget is saturated")

const defaultSlots = 8
const defaultAdmissionWait = 500 * time.Millisecond

var globalSlots = make(chan struct{}, defaultSlots)

type transport struct {
	base  http.RoundTripper
	slots chan struct{}
	wait  time.Duration
}

type releaseBody struct {
	body    interface {
		Read([]byte) (int, error)
		Close() error
	}
	release func()
	once    sync.Once
}

func (b *releaseBody) Read(p []byte) (int, error) { return b.body.Read(p) }

func (b *releaseBody) Close() error {
	err := b.body.Close()
	b.once.Do(b.release)
	return err
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	timer := time.NewTimer(t.wait)
	defer timer.Stop()

	select {
	case t.slots <- struct{}{}:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-timer.C:
		return nil, ErrSaturated
	}

	released := false
	release := func() {
		if released {
			return
		}
		released = true
		<-t.slots
	}

	resp, err := base.RoundTrip(req)
	if err != nil {
		release()
		return nil, err
	}
	if resp.Body == nil {
		release()
		return resp, nil
	}
	resp.Body = &releaseBody{body: resp.Body, release: release}
	return resp, nil
}

func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &transport{
			base:  http.DefaultTransport,
			slots: globalSlots,
			wait:  defaultAdmissionWait,
		},
	}
}

func IsSaturated(err error) bool { return errors.Is(err, ErrSaturated) }

func WithRequestContext(req *http.Request, ctx context.Context) *http.Request {
	return req.Clone(ctx)
}
