package database

import (
	"context"
	"database/sql"

	"github.com/soltros/Supernova/internal/models"
)

// CreateUser creates a new user in the database
func (r *Repository) CreateUser(ctx context.Context, username, passwordHash string) (*models.User, error) {
	id := generateUUID()
	query := `INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, id, username, passwordHash)
	if err != nil {
		return nil, err
	}

	return &models.User{
		ID:       id,
		Username: username,
	}, nil
}

// GetUserByUsername retrieves a user and their password hash for authentication
func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*models.User, string, error) {
	query := `SELECT id, username, password_hash, created_at FROM users WHERE username = ?`
	
	var user models.User
	var hash string
	
	err := r.db.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username, &hash, &user.CreatedAt)
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
	query := `SELECT id, username, created_at FROM users WHERE id = ?`
	
	var user models.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Username, &user.CreatedAt)
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
