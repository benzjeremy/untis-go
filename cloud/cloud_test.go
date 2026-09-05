package cloud

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/benzjeremy/untis-go/db"
)

func TestCloudExportImport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "untis-cloud-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()

	// Add sample data
	_ = database.SetSetting("theme", "nord")
	_ = database.SetSetting("selected_class_name", "IT12A")
	_ = database.SetSetting("ms_user_email", "student@school.edu")

	p := &db.Profile{
		ID:       "prof_test_1",
		Name:     "Test Profile",
		School:   "test-school",
		Server:   "https://test.webuntis.com",
		Username: "student1",
		IsActive: true,
	}
	_ = database.SaveProfile(p)

	h := &db.Homework{
		ID:          "hw_1",
		ProfileID:   "prof_test_1",
		Subject:     "Mathe",
		Description: "Aufgaben 1-5",
		DueDate:     "2026-09-10",
		Completed:   false,
	}
	_ = database.CreateHomework(h)

	// Test Export
	data, err := ExportLocalConfig(database)
	if err != nil {
		t.Fatalf("ExportLocalConfig failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatalf("ExportLocalConfig returned empty bytes")
	}

	// Create second DB to test Import
	dbPath2 := filepath.Join(tmpDir, "test2.db")
	database2, err := db.InitDB(dbPath2)
	if err != nil {
		t.Fatalf("InitDB 2 failed: %v", err)
	}
	defer database2.Close()

	if err := ImportConfigToLocal(database2, data); err != nil {
		t.Fatalf("ImportConfigToLocal failed: %v", err)
	}

	// Verify imported data
	if val := database2.GetSetting("theme", ""); val != "nord" {
		t.Errorf("Expected imported theme 'nord', got '%s'", val)
	}
	if val := database2.GetSetting("selected_class_name", ""); val != "IT12A" {
		t.Errorf("Expected imported class 'IT12A', got '%s'", val)
	}

	importedProf, err := database2.GetProfile("prof_test_1")
	if err != nil || importedProf == nil {
		t.Errorf("Expected profile 'prof_test_1' to be imported, err: %v", err)
	} else if importedProf.School != "test-school" {
		t.Errorf("Expected profile school 'test-school', got '%s'", importedProf.School)
	}

	hwList, err := database2.GetHomeworks("prof_test_1")
	if err != nil || len(hwList) != 1 {
		t.Errorf("Expected 1 homework imported, got %d, err: %v", len(hwList), err)
	} else if hwList[0].Subject != "Mathe" {
		t.Errorf("Expected homework subject 'Mathe', got '%s'", hwList[0].Subject)
	}
}

func TestPKCEGeneration(t *testing.T) {
	v, c := generatePKCE()
	if len(v) == 0 || len(c) == 0 {
		t.Errorf("generatePKCE produced empty verifier or challenge")
	}
}
