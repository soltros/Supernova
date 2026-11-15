package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadDoesNotOverwriteConcurrentDestination(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dest := filepath.Join(t.TempDir(), "song.mp3")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := os.WriteFile(dest, []byte("keep me"), 0600); err != nil {
			t.Error(err)
		}
		w.Write([]byte("download"))
	}))
	defer server.Close()
	if err := saveConfig(&Config{URL: server.URL, Token: "test"}); err != nil {
		t.Fatal(err)
	}
	if err := downloadFile("/song", dest); err == nil {
		t.Fatal("expected destination conflict")
	}
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != "keep me" {
		t.Fatalf("destination overwritten: %q %v", data, err)
	}
	partials, err := filepath.Glob(filepath.Join(filepath.Dir(dest), ".*.partial-*"))
	if err != nil || len(partials) != 0 {
		t.Fatalf("temporary files retained: %v %v", partials, err)
	}
}
