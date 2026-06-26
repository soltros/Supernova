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
