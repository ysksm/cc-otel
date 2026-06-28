package store

import (
	"errors"
	"testing"
)

func TestAccounts(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if n, _ := s.CountAccounts(); n != 0 {
		t.Fatalf("expected 0 accounts, got %d", n)
	}

	admin, err := s.CreateAccount("Admin@Local", "hash1", "admin")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if admin.Email != "admin@local" || admin.Role != "admin" {
		t.Fatalf("admin normalized wrong: %+v", admin)
	}
	if _, err := s.CreateAccount("dev@local", "hash2", "bogus"); err != nil {
		t.Fatalf("create member: %v", err)
	}

	got, err := s.GetAccountByEmail("admin@local")
	if err != nil || got.ID != admin.ID || got.PasswordHash != "hash1" {
		t.Fatalf("get by email: %+v err=%v", got, err)
	}
	byID, err := s.GetAccountByID(admin.ID)
	if err != nil || byID.Email != "admin@local" {
		t.Fatalf("get by id: %+v err=%v", byID, err)
	}

	// member role defaulted from bogus.
	mem, _ := s.GetAccountByEmail("dev@local")
	if mem.Role != "member" {
		t.Fatalf("expected member role, got %q", mem.Role)
	}

	// Unknown email -> ErrNotFound.
	if _, err := s.GetAccountByEmail("nobody@local"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Duplicate email rejected.
	if _, err := s.CreateAccount("admin@local", "h", "admin"); err == nil {
		t.Fatal("expected duplicate email error")
	}

	all, err := s.ListAccounts()
	if err != nil || len(all) != 2 {
		t.Fatalf("list accounts: %d err=%v", len(all), err)
	}
	if n, _ := s.CountAccounts(); n != 2 {
		t.Fatalf("expected 2 accounts, got %d", n)
	}
}
