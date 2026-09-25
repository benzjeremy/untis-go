package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCrypto(t *testing.T) {
	plain := "SecretPass123!_@#"
	encrypted, err := EncryptPassword(plain)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if encrypted == plain {
		t.Fatalf("ciphertext must not match plaintext")
	}

	decrypted, err := DecryptPassword(encrypted)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decrypted != plain {
		t.Fatalf("expected '%s', got '%s'", plain, decrypted)
	}

	// Empty string test
	encEmpty, err := EncryptPassword("")
	if err != nil || encEmpty != "" {
		t.Fatalf("encrypting empty string should return empty string, got: %s", encEmpty)
	}
	decEmpty, err := DecryptPassword("")
	if err != nil || decEmpty != "" {
		t.Fatalf("decrypting empty string should return empty string, got: %s", decEmpty)
	}
}

func TestDatabaseOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "untis_db_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test_untis.db")
	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()

	// 1. Profile tests
	p1 := &Profile{
		ID:       "1",
		Name:     "Test Student",
		School:   "test-schule",
		Server:   "https://untis.testschule.de",
		Username: "student1",
		Password: "supersecretpassword1",
		IsActive: true,
	}

	if err := database.SaveProfile(p1); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}

	p2 := &Profile{
		ID:       "2",
		Name:     "Test Teacher",
		School:   "test-schule",
		Server:   "https://untis.testschule.de",
		Username: "teacher1",
		Password: "supersecretpassword2",
		IsActive: false,
	}

	if err := database.SaveProfile(p2); err != nil {
		t.Fatalf("SaveProfile p2 failed: %v", err)
	}

	profiles, err := database.GetProfiles()
	if err != nil {
		t.Fatalf("GetProfiles failed: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}

	// Check password was decrypted in memory
	active, err := database.GetActiveProfile()
	if err != nil {
		t.Fatalf("GetActiveProfile failed: %v", err)
	}
	if active.ID != "1" || active.Password != "supersecretpassword1" {
		t.Fatalf("active profile mismatch: %+v", active)
	}

	// Verify that password in SQLite table is encrypted and NOT plain text!
	var rawEncrypted string
	err = database.db.QueryRow(`SELECT encrypted_password FROM profiles WHERE id = '1'`).Scan(&rawEncrypted)
	if err != nil {
		t.Fatalf("querying raw password failed: %v", err)
	}
	if rawEncrypted == "supersecretpassword1" || len(rawEncrypted) < 20 {
		t.Fatalf("database contains unencrypted or invalid password: %s", rawEncrypted)
	}

	// Switch profile test
	if err := database.SetActiveProfile("2"); err != nil {
		t.Fatalf("SetActiveProfile failed: %v", err)
	}
	active2, err := database.GetActiveProfile()
	if err != nil || active2.ID != "2" || active2.Password != "supersecretpassword2" {
		t.Fatalf("profile switch failed: %+v", active2)
	}

	// 2. Class tests
	classes := []Class{
		{ID: 101, School: "test-school-xyz", Name: "ITT125", LongName: "IT-Systemelektroniker 125", Active: true},
		{ID: 102, School: "test-school-xyz", Name: "FIA125", LongName: "Fachinformatiker 125", Active: true},
	}
	if err := database.SaveClasses("test-school-xyz", classes); err != nil {
		t.Fatalf("SaveClasses failed: %v", err)
	}

	fetchedClasses, err := database.GetClasses("test-school-xyz")
	if err != nil || len(fetchedClasses) != 2 {
		t.Fatalf("GetClasses failed, got %d classes", len(fetchedClasses))
	}

	// 3. Timetable Cache tests (< 1ms benchmark)
	start := time.Now()
	testJSON := `[{"id": 1, "subject": "LF09", "teacher": "KHN"}]`
	if err := database.SaveTimetableCache(101, "2026-09-09", testJSON); err != nil {
		t.Fatalf("SaveTimetableCache failed: %v", err)
	}

	cachedJSON, _, found, err := database.GetTimetableCache(101, "2026-09-09")
	elapsed := time.Since(start)
	if err != nil || !found || cachedJSON != testJSON {
		t.Fatalf("GetTimetableCache failed: found=%v, json=%s", found, cachedJSON)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("Cache retrieval took too long: %v", elapsed)
	}

	// 4. Settings tests
	if err := database.SetSetting("theme", "light"); err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}
	if val := database.GetSetting("theme", "dark"); val != "light" {
		t.Fatalf("expected theme 'light', got '%s'", val)
	}
}

func TestMicrosoftSettingsPurgedAndBlocked(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "untis-db-ms-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test_ms.db")
	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()

	// Setting ms_ keys must be rejected/ignored
	if err := database.SetSetting("ms_access_token", "secret123"); err != nil {
		t.Fatalf("SetSetting returned unexpected error: %v", err)
	}
	if val := database.GetSetting("ms_access_token", ""); val != "" {
		t.Fatalf("expected empty string for ms_access_token, got '%s'", val)
	}

	// Directly insert ms_ key into table to simulate legacy DB
	rawDB := database.GetUnderlyingDB()
	_, err = rawDB.Exec("INSERT INTO settings (key, value) VALUES ('ms_user_email', 'leak@example.com')")
	if err != nil {
		t.Fatalf("failed to insert raw legacy setting: %v", err)
	}

	// GetAllSettings must filter it out
	all, err := database.GetAllSettings()
	if err != nil {
		t.Fatalf("GetAllSettings failed: %v", err)
	}
	if _, exists := all["ms_user_email"]; exists {
		t.Fatalf("GetAllSettings leaked legacy ms_ key: %v", all)
	}

	// Re-initializing DB must purge any ms_ keys
	database.Close()
	dbReopened, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("reopening InitDB failed: %v", err)
	}
	defer dbReopened.Close()

	var count int
	err = dbReopened.GetUnderlyingDB().QueryRow("SELECT COUNT(*) FROM settings WHERE key LIKE 'ms_%'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count ms_ keys: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 ms_ keys in settings table after InitDB, found %d", count)
	}
}
