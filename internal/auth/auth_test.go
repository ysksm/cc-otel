package auth

import (
	"testing"
	"time"
)

func TestPasswordHashing(t *testing.T) {
	h, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !CheckPassword(h, "hunter2") {
		t.Fatal("correct password rejected")
	}
	if CheckPassword(h, "wrong") {
		t.Fatal("wrong password accepted")
	}
}

func TestSessionSignVerify(t *testing.T) {
	secret := []byte("test-secret")
	tok := SignSession(secret, "user-1", time.Now().Add(time.Hour))
	id, ok := VerifySession(secret, tok)
	if !ok || id != "user-1" {
		t.Fatalf("valid session failed: id=%s ok=%v", id, ok)
	}
	// Wrong secret rejected.
	if _, ok := VerifySession([]byte("other"), tok); ok {
		t.Fatal("session verified with wrong secret")
	}
	// Expired rejected.
	expired := SignSession(secret, "user-1", time.Now().Add(-time.Minute))
	if _, ok := VerifySession(secret, expired); ok {
		t.Fatal("expired session accepted")
	}
	// Tampered token rejected.
	if _, ok := VerifySession(secret, tok+"x"); ok {
		t.Fatal("tampered token accepted")
	}
}
