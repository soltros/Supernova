package database

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"errors"

	"github.com/soltros/Supernova/internal/models"
)

// CreateUser creates a new user in the database
func (r *Repository) CreateUser(ctx context.Context, username, passwordHash string) (*models.User, error) {
	return r.createUser(ctx, username, passwordHash, "", "", false)
}

var ErrInviteRequired = errors.New("a valid invite code is required")

// RegisterUser validates the invite and creates the account in one serialized operation.
// Only the first account can bootstrap without an invite and receives admin rights.
func (r *Repository) RegisterUser(ctx context.Context, username, hash, invite, configuredInvite string) (*models.User, error) {
	return r.createUser(ctx, username, hash, invite, configuredInvite, true)
}

func (r *Repository) createUser(ctx context.Context, username, hash, invite, configuredInvite string, checkInvite bool) (*models.User, error) {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Take the write lock before checking bootstrap state, including across processes.
	if _, err := tx.ExecContext(ctx, `UPDATE users SET is_admin = is_admin WHERE 0`); err != nil {
		return nil, err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return nil, err
	}
	if checkInvite && count > 0 {
		actual, expected := sha256.Sum256([]byte(invite)), sha256.Sum256([]byte(configuredInvite))
		if configuredInvite == "" || subtle.ConstantTimeCompare(actual[:], expected[:]) != 1 {
			return nil, ErrInviteRequired
		}
	}
	user := &models.User{ID: generateUUID(), Username: username, IsAdmin: count == 0}
	if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, username, password_hash, is_admin) VALUES (?, ?, ?, ?)`, user.ID, username, hash, user.IsAdmin); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByUsername retrieves a user and their password hash for authentication
func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*models.User, string, error) {
	query := `SELECT id, username, password_hash, created_at, is_admin FROM users WHERE username = ?`

	var user models.User
	var hash string

	err := r.db.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username, &hash, &user.CreatedAt, &user.IsAdmin)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return &user, hash, nil
}

// GetUserByID checks if a user exists by ID
func (r *Repository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	query := `SELECT id, username, created_at, is_admin FROM users WHERE id = ?`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Username, &user.CreatedAt, &user.IsAdmin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetSubsonicPassword retrieves the encrypted subsonic password for a user
func (r *Repository) GetSubsonicPassword(ctx context.Context, username string) (string, error) {
	query := `SELECT subsonic_password FROM users WHERE username = ?`
	var encPass sql.NullString
	err := r.db.QueryRowContext(ctx, query, username).Scan(&encPass)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return encPass.String, nil
}

// SetSubsonicPassword saves the symmetrically encrypted password for subsonic auth
func (r *Repository) SetSubsonicPassword(ctx context.Context, username, encryptedPassword string) error {
	query := `UPDATE users SET subsonic_password = ? WHERE username = ?`
	_, err := r.db.ExecContext(ctx, query, encryptedPassword, username)
	return err
}
