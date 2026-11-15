package api

import (
	"fmt"
	"testing"
	"time"
)

func TestAttemptLimiterBoundAndExpiry(t *testing.T) {
	l := &attemptLimiter{entries: map[string]attemptState{}}
	for i := 0; i < maxAuthAttempts+100; i++ {
		l.failed(fmt.Sprint(i))
	}
	if len(l.entries) != maxAuthAttempts {
		t.Fatalf("unbounded entries: %d", len(l.entries))
	}
	if l.allowed("new") {
		t.Fatal("new key admitted at capacity")
	}
	if !l.allowed("0") {
		t.Fatal("existing key rejected")
	}
	for k, v := range l.entries {
		v.first = time.Now().Add(-11 * time.Minute)
		l.entries[k] = v
	}
	l.lastCleanup = time.Time{}
	if !l.allowed("new") || len(l.entries) != 0 {
		t.Fatal("expired entries retained")
	}
	for i := 0; i < 5; i++ {
		l.failed("blocked")
	}
	if l.allowed("blocked") {
		t.Fatal("failed attempts not blocked")
	}
	l.success("blocked")
	if !l.allowed("blocked") {
		t.Fatal("success did not reset limiter")
	}
}
