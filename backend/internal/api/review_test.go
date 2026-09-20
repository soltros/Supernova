package api

import (
	"context"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/soltros/Supernova/internal/database"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuthInviteAdminAndPluginBoundary(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("REGISTRATION_INVITE_CODE", "invite")
	db, err := database.Init(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	server := NewServer(repo, nil, nil, nil, nil)
	server.mux.HandleFunc("POST /api/plugins/deduper/run", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) })
	request := func(method, path, body, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	register := func(name, invite string) AuthResponse {
		w := request("POST", "/api/auth/register", `{"username":"`+name+`","password":"password","invite_code":"`+invite+`"}`, "")
		if w.Code != 200 {
			t.Fatalf("registration %s: %d %s", name, w.Code, w.Body)
		}
		var result AuthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	admin := register("admin", "")
	if !admin.User.IsAdmin {
		t.Fatal("bootstrap not admin")
	}
	w := request("POST", "/api/auth/register", `{"username":"intruder","password":"password"}`, "")
	if w.Code != 403 {
		t.Fatalf("no invite: %d", w.Code)
	}
	member := register("member", "invite")
	if member.User.IsAdmin {
		t.Fatal("member is admin")
	}
	for _, tc := range []struct {
		token string
		want  int
	}{{"", 401}, {member.Token, 403}, {admin.Token, 202}} {
		w := request("POST", "/api/plugins/deduper/run", "", tc.token)
		if w.Code != tc.want {
			t.Fatalf("plugin auth: got %d want %d", w.Code, tc.want)
		}
	}
	for _, claims := range []jwt.MapClaims{{"user_id": member.User.ID}, {"user_id": member.User.ID, "exp": time.Now().Add(-time.Hour).Unix()}} {
		token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(strings.Repeat("s", 32)))
		if w := request("GET", "/api/hearts", "", token); w.Code != 401 {
			t.Fatalf("invalid claims accepted: %d", w.Code)
		}
	}
	if w := request("POST", "/api/scan?token="+admin.Token, "", ""); w.Code != 401 {
		t.Fatalf("query token permits mutation: %d", w.Code)
	}
	if _, err := db.ExecContext(context.Background(), "DELETE FROM users WHERE id=?", member.User.ID); err != nil {
		t.Fatal(err)
	}
	if w := request("GET", "/api/plugins/podcasts/subscriptions", "", member.Token); w.Code != 401 {
		t.Fatalf("deleted user accepted: %d", w.Code)
	}
}
