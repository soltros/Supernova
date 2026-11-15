package resourcebudget

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

var ErrSaturated = errors.New("outbound request budget is saturated")

var ErrResponseTooLarge = errors.New("upstream metadata response exceeds 8 MiB")

const maxResponseBytes int64 = 8 << 20

type boundedBody struct {
	io.ReadCloser
	remaining int64
}

func (b *boundedBody) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if b.remaining == 0 {
		var extra [1]byte
		n, err := b.ReadCloser.Read(extra[:])
		if n > 0 {
			return 0, ErrResponseTooLarge
		}
		return 0, err
	}
	if int64(len(p)) > b.remaining {
		p = p[:int(b.remaining)]
	}
	n, err := b.ReadCloser.Read(p)
	b.remaining -= int64(n)
	return n, err
}

const defaultSlots = 8
const defaultAdmissionWait = 500 * time.Millisecond

var globalSlots = make(chan struct{}, defaultSlots)

type transport struct {
	base  http.RoundTripper
	slots chan struct{}
	wait  time.Duration
}

type releaseBody struct {
	body interface {
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
	if resp.ContentLength > maxResponseBytes {
		resp.Body.Close()
		release()
		return nil, ErrResponseTooLarge
	}
	resp.Body = &releaseBody{body: &boundedBody{ReadCloser: resp.Body, remaining: maxResponseBytes}, release: release}
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
