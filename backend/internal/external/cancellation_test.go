package external

import (
	"context"
	"errors"
	"github.com/soltros/Supernova/internal/models"
	"net/http"
	"testing"
	"time"
)

type cancelTransport func(*http.Request) (*http.Response, error)

func (f cancelTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestLastFmContextCancelsWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewLastFmClient("key", "secret")
	c.client = &http.Client{Transport: cancelTransport(func(r *http.Request) (*http.Response, error) {
		if r.Context().Err() == nil {
			t.Error("request context missing")
		}
		return nil, r.Context().Err()
	})}
	if _, err := c.WithContext(ctx).GetArtistInfo("artist"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if err := c.WithContext(ctx).Scrobble("session", "artist", "song", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("scrobble cancellation: %v", err)
	}
}
func TestMusicBrainzCancellationWhileWaitingForRateLimit(t *testing.T) {
	c := NewMusicBrainzClient("test", "1", "test@example.test")
	c.gate <- struct{}{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	started := time.Now()
	err := c.EnhanceMetadataContext(ctx, &models.TrackMetadata{Title: "title", Artist: "artist"})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > time.Second {
		t.Fatalf("rate-limit wait ignored cancellation: %v", err)
	}
}
