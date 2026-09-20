package authn

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/soltros/Supernova/internal/database"
	"github.com/soltros/Supernova/internal/models"
	"net/http"
	"os"
	"strings"
)

func Secret() ([]byte, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must contain at least 32 characters")
	}
	return secret, nil
}

func Authenticate(r *http.Request, repo *database.Repository, allowQuery bool) (*models.User, error) {
	secret, err := Secret()
	if err != nil {
		return nil, err
	}
	value := ""
	if header := r.Header.Get("Authorization"); header != "" {
		fields := strings.Fields(header)
		if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
			return nil, errors.New("invalid authorization header")
		}
		value = fields[1]
	} else if allowQuery {
		value = r.URL.Query().Get("token")
	}
	if value == "" {
		return nil, errors.New("missing token")
	}
	token, err := jwt.Parse(value, func(*jwt.Token) (any, error) { return secret, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	id, ok := claims["user_id"].(string)
	if !ok || id == "" {
		return nil, errors.New("invalid user")
	}
	user, err := repo.GetUserByID(r.Context(), id)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}
