package store

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/ysksm/cc-otel/internal/model"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// Bootstrap ensures a default workspace, project and API key exist. If a key is
// created, its plaintext token is returned (only available at creation time) and
// also written to <dataDir>/dev-api-key.txt for local convenience.
func (s *Store) Bootstrap(devToken string) (ws model.Workspace, proj model.Project, token string, err error) {
	ws, err = s.ensureWorkspace("Local", "local")
	if err != nil {
		return
	}
	proj, err = s.ensureProject(ws.ID, "Default", "default")
	if err != nil {
		return
	}

	var keyCount int
	if err = s.db.QueryRow(`SELECT count(*) FROM api_keys WHERE workspace_id = ?`, ws.ID).Scan(&keyCount); err != nil {
		return
	}
	if keyCount == 0 {
		var key model.APIKey
		key, err = s.CreateAPIKey(ws.ID, "Local dev key", devToken)
		if err != nil {
			return
		}
		token = key.Token
		_ = os.WriteFile(filepath.Join(s.dataDir, "dev-api-key.txt"), []byte(token+"\n"), 0o600)
	}
	return
}

func (s *Store) ensureWorkspace(name, slug string) (model.Workspace, error) {
	var w model.Workspace
	err := s.db.QueryRow(`SELECT id, name, slug, created_at FROM workspaces WHERE slug = ?`, slug).
		Scan(&w.ID, &w.Name, &w.Slug, &w.CreatedAt)
	if err == nil {
		return w, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return w, err
	}
	w = model.Workspace{ID: newID(), Name: name, Slug: slug, CreatedAt: time.Now()}
	if _, err := s.db.Exec(`INSERT INTO workspaces (id, name, slug) VALUES (?, ?, ?)`, w.ID, w.Name, w.Slug); err != nil {
		return w, err
	}
	return w, nil
}

func (s *Store) ensureProject(workspaceID, name, slug string) (model.Project, error) {
	p, err := s.GetProjectBySlug(workspaceID, slug)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return p, err
	}
	p = model.Project{ID: newID(), WorkspaceID: workspaceID, Name: name, Slug: slug, CreatedAt: time.Now()}
	if _, err := s.db.Exec(`INSERT INTO projects (id, workspace_id, name, slug) VALUES (?, ?, ?, ?)`,
		p.ID, p.WorkspaceID, p.Name, p.Slug); err != nil {
		return p, err
	}
	return p, nil
}

// CreateProject creates a new project in a workspace.
func (s *Store) CreateProject(workspaceID, name, slug string) (model.Project, error) {
	return s.ensureProject(workspaceID, name, slug)
}

// ListProjects returns all projects in a workspace, newest first.
func (s *Store) ListProjects(workspaceID string) ([]model.Project, error) {
	rows, err := s.db.Query(`SELECT id, workspace_id, name, slug, first_trace_at, created_at
		FROM projects WHERE workspace_id = ? ORDER BY created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Project
	for rows.Next() {
		var p model.Project
		var firstTrace sql.NullTime
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Slug, &firstTrace, &p.CreatedAt); err != nil {
			return nil, err
		}
		if firstTrace.Valid {
			p.FirstTraceAt = &firstTrace.Time
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetProjectBySlug looks up a project by workspace + slug.
func (s *Store) GetProjectBySlug(workspaceID, slug string) (model.Project, error) {
	var p model.Project
	var firstTrace sql.NullTime
	err := s.db.QueryRow(`SELECT id, workspace_id, name, slug, first_trace_at, created_at
		FROM projects WHERE workspace_id = ? AND slug = ?`, workspaceID, slug).
		Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Slug, &firstTrace, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	if firstTrace.Valid {
		p.FirstTraceAt = &firstTrace.Time
	}
	return p, nil
}

// GetProjectByID looks up a project by id.
func (s *Store) GetProjectByID(id string) (model.Project, error) {
	var p model.Project
	var firstTrace sql.NullTime
	err := s.db.QueryRow(`SELECT id, workspace_id, name, slug, first_trace_at, created_at
		FROM projects WHERE id = ?`, id).
		Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Slug, &firstTrace, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	if firstTrace.Valid {
		p.FirstTraceAt = &firstTrace.Time
	}
	return p, nil
}

// CreateAPIKey creates an API key for a workspace. If token is empty a random
// one is generated. The plaintext token is returned on the result.
func (s *Store) CreateAPIKey(workspaceID, name, token string) (model.APIKey, error) {
	var hash string
	if token == "" {
		token, hash = newToken()
	} else {
		hash = hashToken(token)
	}
	preview := token
	if len(preview) > 14 {
		preview = preview[:14]
	}
	k := model.APIKey{
		ID: newID(), WorkspaceID: workspaceID, Name: name,
		TokenHash: hash, TokenPreview: preview, Token: token, CreatedAt: time.Now(),
	}
	if _, err := s.db.Exec(`INSERT INTO api_keys (id, workspace_id, name, token_hash, token_preview)
		VALUES (?, ?, ?, ?, ?)`, k.ID, k.WorkspaceID, k.Name, k.TokenHash, k.TokenPreview); err != nil {
		return model.APIKey{}, err
	}
	return k, nil
}

// ListAPIKeys returns keys for a workspace (without plaintext tokens).
func (s *Store) ListAPIKeys(workspaceID string) ([]model.APIKey, error) {
	rows, err := s.db.Query(`SELECT id, workspace_id, name, token_preview, created_at, last_used_at
		FROM api_keys WHERE workspace_id = ? ORDER BY created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.APIKey
	for rows.Next() {
		var k model.APIKey
		var name sql.NullString
		var lastUsed sql.NullTime
		if err := rows.Scan(&k.ID, &k.WorkspaceID, &name, &k.TokenPreview, &k.CreatedAt, &lastUsed); err != nil {
			return nil, err
		}
		k.Name = name.String
		if lastUsed.Valid {
			k.LastUsedAt = &lastUsed.Time
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// ResolveAPIKey validates a plaintext token and returns the owning workspace and
// key id. It returns ErrNotFound for unknown/invalid tokens.
func (s *Store) ResolveAPIKey(token string) (workspaceID, keyID string, err error) {
	hash := hashToken(token)
	err = s.db.QueryRow(`SELECT workspace_id, id FROM api_keys WHERE token_hash = ?`, hash).
		Scan(&workspaceID, &keyID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return workspaceID, keyID, err
}

// TouchAPIKey records that a key was just used.
func (s *Store) TouchAPIKey(keyID string) {
	_, _ = s.db.Exec(`UPDATE api_keys SET last_used_at = now() WHERE id = ?`, keyID)
}
