package api

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/soltros/Supernova/internal/authn"
	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/models"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func getJWTSecret() []byte {
	secret, err := authn.Secret()
	if err != nil {
		log.Fatal(err)
	}
	return secret
}

type contextKey string

const userIDKey contextKey = "user_id"

type AuthRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	InviteCode string `json:"invite_code"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

func (s *Server) handleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		req.Username = strings.TrimSpace(req.Username)
		if len(req.Username) < 3 || len(req.Username) > 64 || len(req.Password) < 6 || len(req.Password) > 72 {
			http.Error(w, "username must be 3–64 bytes and password 6–72 bytes", http.StatusBadRequest)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}

		user, err := s.repo.RegisterUser(r.Context(), req.Username, string(hash), req.InviteCode, os.Getenv("REGISTRATION_INVITE_CODE"))
		if errors.Is(err, database.ErrInviteRequired) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if err != nil {
			http.Error(w, "username already exists", http.StatusConflict)
			return
		}

		if encPass, err := EncryptPassword(req.Password, getJWTSecret()); err == nil {
			_ = s.repo.SetSubsonicPassword(r.Context(), user.Username, encPass)
		}

		token, err := generateJWT(user.ID)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(AuthResponse{
			Token: token,
			User:  user,
		})
	}
}

func (s *Server) handleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		user, hash, err := s.repo.GetUserByUsername(r.Context(), req.Username)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		if user == nil {
			http.Error(w, "invalid username or password", http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
			http.Error(w, "invalid username or password", http.StatusUnauthorized)
			return
		}

		// Ensure the subsonic password is encrypted and saved
		if encPass, err := EncryptPassword(req.Password, getJWTSecret()); err == nil {
			_ = s.repo.SetSubsonicPassword(r.Context(), user.Username, encPass)
		}

		token, err := generateJWT(user.ID)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(AuthResponse{
			Token: token,
			User:  user,
		})
	}
}

func generateJWT(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 day expiration
	})
	return token.SignedString(getJWTSecret())
}

// requireAuth validates the token against the current account on every request.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowQuery := r.Method == http.MethodGet || r.Method == http.MethodHead
		allowQuery = allowQuery && (strings.HasPrefix(r.URL.Path, "/api/stream/") || strings.HasPrefix(r.URL.Path, "/api/download/"))
		user, err := authn.Authenticate(r, s.repo, allowQuery)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, user.ID)
		ctx = context.WithValue(ctx, contextKey("user"), user)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(contextKey("user")).(*models.User)
		if !user.IsAdmin {
			http.Error(w, "administrator access required", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}
