package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.1.0", "1.0.0", 1},
		{"1.0.0", "v1.1.0", -1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.0.1", "1.0.0", 1},
		{"v1.1.0-beta.1", "1.0.0", 1},
	}

	for _, tt := range tests {
		got := CompareVersions(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("CompareVersions(%q, %q) = %d; expected %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestExtractFromTarGz(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("binary-content-test")
	hdr := &tar.Header{
		Name: "untis-go",
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	_ = tw.Close()
	_ = gw.Close()

	extracted, err := extractFromTarGz(buf.Bytes())
	if err != nil {
		t.Fatalf("extractFromTarGz failed: %v", err)
	}
	if string(extracted) != string(content) {
		t.Fatalf("expected %q, got %q", content, extracted)
	}
}

func TestExtractFromZip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	content := []byte("binary-content-zip-test")
	f, err := zw.Create("untis-go.exe")
	if err != nil {
		t.Fatalf("Create in zip failed: %v", err)
	}
	if _, err := f.Write(content); err != nil {
		t.Fatalf("Write in zip failed: %v", err)
	}
	_ = zw.Close()

	extracted, err := extractFromZip(buf.Bytes())
	if err != nil {
		t.Fatalf("extractFromZip failed: %v", err)
	}
	if string(extracted) != string(content) {
		t.Fatalf("expected %q, got %q", content, extracted)
	}
}
