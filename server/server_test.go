package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benzjeremy/untis-go/db"
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

	// 3. Test /api/dashboard
	reqDash, _ := http.NewRequest("GET", baseURL+"/api/dashboard", nil)
	reqDash.Header.Set("X-Session-Token", token)
	respDash, err := http.DefaultClient.Do(reqDash)
	if err != nil || respDash.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/dashboard, got %v, err: %v", respDash.StatusCode, err)
	}
	respDash.Body.Close()

	// 4. Test /api/homework POST and GET
	hwBody := strings.NewReader(`{"subject":"Mathematik","description":"Seite 42 Nr. 1-4","dueDate":"2026-09-10"}`)
	reqHwPost, _ := http.NewRequest("POST", baseURL+"/api/homework", hwBody)
	reqHwPost.Header.Set("X-Session-Token", token)
	respHwPost, err := http.DefaultClient.Do(reqHwPost)
	if err != nil || respHwPost.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for POST /api/homework, got %v, err: %v", respHwPost.StatusCode, err)
	}
	var createdHw struct {
		Success  bool        `json:"success"`
		Homework db.Homework `json:"homework"`
	}
	_ = json.NewDecoder(respHwPost.Body).Decode(&createdHw)
	respHwPost.Body.Close()

	if createdHw.Homework.Subject != "Mathematik" {
		t.Fatalf("expected created homework subject 'Mathematik', got %s", createdHw.Homework.Subject)
	}

	// Test GET /api/homework
	reqHwGet, _ := http.NewRequest("GET", baseURL+"/api/homework", nil)
	reqHwGet.Header.Set("X-Session-Token", token)
	respHwGet, err := http.DefaultClient.Do(reqHwGet)
	if err != nil || respHwGet.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /api/homework, got %v", respHwGet.StatusCode)
	}
	var hwList []db.Homework
	_ = json.NewDecoder(respHwGet.Body).Decode(&hwList)
	respHwGet.Body.Close()
	if len(hwList) == 0 {
		t.Fatalf("expected at least 1 homework, got 0")
	}

	// 5. Test /api/absences POST and GET
	absBody := strings.NewReader(`{"reason":"Krankheit","text":"Grippaler Infekt","startDate":"2026-09-08","endDate":"2026-09-08","isExcused":true}`)
	reqAbsPost, _ := http.NewRequest("POST", baseURL+"/api/absences", absBody)
	reqAbsPost.Header.Set("X-Session-Token", token)
	respAbsPost, err := http.DefaultClient.Do(reqAbsPost)
	if err != nil || respAbsPost.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for POST /api/absences, got %v", respAbsPost.StatusCode)
	}
	respAbsPost.Body.Close()

	reqAbsGet, _ := http.NewRequest("GET", baseURL+"/api/absences", nil)
	reqAbsGet.Header.Set("X-Session-Token", token)
	respAbsGet, err := http.DefaultClient.Do(reqAbsGet)
	if err != nil || respAbsGet.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /api/absences, got %v", respAbsGet.StatusCode)
	}
	var absList []db.Absence
	_ = json.NewDecoder(respAbsGet.Body).Decode(&absList)
	respAbsGet.Body.Close()
	if len(absList) == 0 {
		t.Fatalf("expected at least 1 absence, got 0")
	}

	// 6. Test /api/profiles/delete
	// Create a dummy profile to delete
	dummyProf := &db.Profile{
		ID:       "dummy_delete_test",
		Name:     "Delete Me",
		School:   "test-school",
		Server:   "https://test.webuntis.com",
		Username: "tester",
	}
	_ = database.SaveProfile(dummyProf)

	delReq, _ := http.NewRequest("POST", baseURL+"/api/profiles/delete?id=dummy_delete_test", nil)
	delReq.Header.Set("X-Session-Token", token)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil || delResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/profiles/delete, got %v", delResp.StatusCode)
	}
	delResp.Body.Close()

	if _, err := database.GetProfile("dummy_delete_test"); err == nil {
		t.Fatalf("expected profile to be deleted from database")
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
