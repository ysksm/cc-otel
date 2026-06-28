package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/ysksm/cc-otel/internal/model"
)

const accountCols = `id, email, password_hash, role, created_at`

func scanAccount(sc interface{ Scan(...any) error }) (model.Account, error) {
	var a model.Account
	err := sc.Scan(&a.ID, &a.Email, &a.PasswordHash, &a.Role, &a.CreatedAt)
	return a, err
}

// CreateAccount inserts an account. email is lowercased.
func (s *Store) CreateAccount(email, passwordHash, role string) (model.Account, error) {
	if role != "admin" {
		role = "member"
	}
	a := model.Account{
		ID: newID(), Email: strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: passwordHash, Role: role, CreatedAt: time.Now(),
	}
	if _, err := s.db.Exec(`INSERT INTO accounts (id, email, password_hash, role) VALUES (?, ?, ?, ?)`,
		a.ID, a.Email, a.PasswordHash, a.Role); err != nil {
		return model.Account{}, err
	}
	return a, nil
}

// GetAccountByEmail looks up an account by email.
func (s *Store) GetAccountByEmail(email string) (model.Account, error) {
	row := s.db.QueryRow(`SELECT `+accountCols+` FROM accounts WHERE email = ?`, strings.ToLower(strings.TrimSpace(email)))
	a, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}

// GetAccountByID looks up an account by id.
func (s *Store) GetAccountByID(id string) (model.Account, error) {
	row := s.db.QueryRow(`SELECT `+accountCols+` FROM accounts WHERE id = ?`, id)
	a, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}

// ListAccounts returns all accounts (without password hashes in callers' output).
func (s *Store) ListAccounts() ([]model.Account, error) {
	rows, err := s.db.Query(`SELECT ` + accountCols + ` FROM accounts ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CountAccounts returns the number of accounts.
func (s *Store) CountAccounts() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT count(*) FROM accounts`).Scan(&n)
	return n, err
}
