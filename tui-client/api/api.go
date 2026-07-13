package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/soltros/Supernova/tui-client/models"
)

var BaseURL = "http://localhost:8080/api"

var JWTToken string

var httpClient = &http.Client{Timeout: 10 * time.Second}

func Login(username, password string) error {
	payload := map[string]string{"username": username, "password": password}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := httpClient.Post(BaseURL+"/auth/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: %s", resp.Status)
	}

	var authResp models.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return err
	}
	JWTToken = authResp.Token
	return nil
}

func Register(username, password string) error {
	payload := map[string]string{"username": username, "password": password}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := httpClient.Post(BaseURL+"/auth/register", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("register failed: %s", resp.Status)
	}

	return nil
}

func Fetch(endpoint string, target interface{}) error {
	req, _ := http.NewRequest("GET", BaseURL+endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+JWTToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return err
	}
	return nil
}

func Post(endpoint string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, _ := http.NewRequest("POST", BaseURL+endpoint, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+JWTToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func HeartToggle(entityType, entityID string) error {
	// Simple POST for now
	payload := map[string]string{"entity_type": entityType, "entity_id": entityID}
	return Post("/hearts", payload)
}
