package database

import (
	"context"
	"testing"
	"time"
)

func TestChangePasswordCredentialsAndSessions(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	u, err := r.CreateUser(ctx, "alice", "old")
	if err != nil {
		t.Fatal(err)
	}
	keep, err := r.CreateSession(ctx, u.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	other, err := r.CreateSession(ctx, u.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.SetSubsonicPassword(ctx, u.Username, "old-encrypted"); err != nil {
		t.Fatal(err)
	}
	if err := r.ChangePassword(ctx, u.ID, "old", "new", "", keep); err != nil {
		t.Fatal(err)
	}
	if !r.SessionActive(ctx, keep, u.ID) || r.SessionActive(ctx, other, u.ID) {
		t.Fatal("incorrect session revocation")
	}
	if err := r.SetSubsonicPasswordForHash(ctx, u.Username, "old", "stale-encrypted"); err != nil {
		t.Fatal(err)
	}
	encrypted, err := r.GetSubsonicPassword(ctx, u.Username)
	if err != nil || encrypted != "" {
		t.Fatalf("stale credential restored: %q %v", encrypted, err)
	}
	if err := r.ChangePassword(ctx, u.ID, "old", "stale", "stale", keep); err == nil {
		t.Fatal("stale password change accepted")
	}
	_, hash, err := r.GetUserByUsername(ctx, u.Username)
	if err != nil || hash != "new" {
		t.Fatalf("password: %q %v", hash, err)
	}
}

func TestChangePasswordRollsBackOnRevocationFailure(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	u, err := r.CreateUser(ctx, "alice", "old")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.SetSubsonicPassword(ctx, u.Username, "old-encrypted"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.db.Exec(`CREATE TRIGGER reject_revocation BEFORE UPDATE ON sessions BEGIN SELECT RAISE(ABORT, 'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateSession(ctx, u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := r.ChangePassword(ctx, u.ID, "old", "new", "new-encrypted", ""); err == nil {
		t.Fatal("expected revocation failure")
	}
	_, hash, err := r.GetUserByUsername(ctx, u.Username)
	if err != nil || hash != "old" {
		t.Fatalf("hash not rolled back: %q %v", hash, err)
	}
	encrypted, err := r.GetSubsonicPassword(ctx, u.Username)
	if err != nil || encrypted != "old-encrypted" {
		t.Fatalf("credential not rolled back: %q %v", encrypted, err)
	}
}

func TestStaleLoginCannotCreateSession(t *testing.T) {
	r := testRepository(t)
	ctx := context.Background()
	u, err := r.CreateUser(ctx, "alice", "old")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ChangePassword(ctx, u.ID, "old", "new", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateSessionForHash(ctx, u.ID, "old", time.Now().Add(time.Hour)); err == nil {
		t.Fatal("stale login accepted")
	}
	if _, err := r.CreateSessionForHash(ctx, u.ID, "new", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
}
