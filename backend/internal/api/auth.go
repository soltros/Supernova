package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret []byte

func getJWTSecret() []byte {
	if jwtSecret != nil {
		return jwtSecret
	}
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		log.Fatal("FATAL: JWT_SECRET env var must be set and at least 32 characters long. Set it before starting the server.")
	}
	jwtSecret = []byte(secret)
	return jwtSecret
}

type contextKey string
const userIDKey contextKey = "user_id"

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
}

func (s *Server) handleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if len(req.Username) < 3 || len(req.Password) < 6 {
			http.Error(w, "username must be 3+ chars and password 6+ chars", http.StatusBadRequest)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}

		user, err := s.repo.CreateUser(r.Context(), req.Username, string(hash))
		if err != nil {
			http.Error(w, "username already exists", http.StatusConflict)
			return
		}

		token, err := generateJWT(user.ID)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(AuthResponse{
			Token: token,
			User: struct {
				ID       string `json:"id"`
				Username string `json:"username"`
			}{
				ID:       user.ID,
				Username: user.Username,
			},
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

		token, err := generateJWT(user.ID)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(AuthResponse{
			Token: token,
			User: struct {
				ID       string `json:"id"`
				Username string `json:"username"`
			}{
				ID:       user.ID,
				Username: user.Username,
			},
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

// requireAuth is a middleware that intercepts protected routes, validates the JWT, and injects the user_id into the context
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := ""
		
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else if r.URL.Query().Get("token") != "" {
			// Fallback to query parameter (required for native <audio> tags connecting to stream endpoints)
			tokenString = r.URL.Query().Get("token")
		}

		if tokenString == "" {
			http.Error(w, "unauthorized - missing token", http.StatusUnauthorized)
			return
		}
		
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "unauthorized - invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "unauthorized - invalid claims", http.StatusUnauthorized)
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			http.Error(w, "unauthorized - invalid user id claim", http.StatusUnauthorized)
			return
		}

		// Double-check the database to completely prevent Phantom Users
		user, err := s.repo.GetUserByID(r.Context(), userID)
		if err != nil || user == nil {
			http.Error(w, "unauthorized - user does not exist", http.StatusUnauthorized)
			return
		}

		// Inject userID into context
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
