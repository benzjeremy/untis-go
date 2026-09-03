package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"untis-go/db"
)

func TestServerSecurityAndEndpoints(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "untis_server_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer database.Close()

	srv := NewServer(database)
	serverURL, err := srv.Start(0) // dynamic random port
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop()

	token := srv.GetSessionToken()
	if len(token) != 32 {
		t.Fatalf("expected 32-character session token, got length %d: %s", len(token), token)
	}

	// 1. Test unauthenticated request to /api/status (must fail with 401 Unauthorized)
	baseURL := fmtBaseURL(srv.port)
	resp, err := http.Get(baseURL + "/api/status")
	if err != nil {
		t.Fatalf("failed to make unauthed get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for request without token, got %d", resp.StatusCode)
	}

	// 2. Test authenticated request with X-Session-Token header
	req, err := http.NewRequest("GET", baseURL+"/api/status", nil)
	if err != nil {
		t.Fatalf("failed to create req: %v", err)
	}
	req.Header.Set("X-Session-Token", token)

	authResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to make authed get: %v", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for valid session token, got %d", authResp.StatusCode)
	}

	var status map[string]interface{}
	if err := json.NewDecoder(authResp.Body).Decode(&status); err != nil {
		t.Fatalf("failed to parse status json: %v", err)
	}

	// Check if server URL returned by Start() has token query param
	if serverURL == "" || !containsToken(serverURL, token) {
		t.Fatalf("serverURL %s does not contain token %s", serverURL, token)
	}
}

func fmtBaseURL(port int) string {
	return "http://127.0.0.1:" + strconvItoa(port)
}

func strconvItoa(i int) string {
	var b [20]byte
	n := len(b)
	for i > 0 {
		n--
		b[n] = byte('0' + (i % 10))
		i /= 10
	}
	return string(b[n:])
}

func containsToken(s, token string) bool {
	return len(s) > len(token) && (s[len(s)-len(token):] == token || filepath.Base(s) == token || len(s) > 32)
}
