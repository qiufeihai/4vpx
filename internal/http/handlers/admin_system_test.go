package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestReadBackupPayloadPrefersUploadedFile(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile("backup_file", "backup.json")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := fileWriter.Write([]byte(`{"from":"file"}`)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.WriteField("backup_json", `{"from":"textarea"}`); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	req := httptest.NewRequest("POST", "/admin/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	payload, err := readBackupPayload(req)
	if err != nil {
		t.Fatalf("readBackupPayload() error = %v", err)
	}
	if got := string(payload); got != `{"from":"file"}` {
		t.Fatalf("readBackupPayload() = %q, want file payload", got)
	}
}

func TestReadBackupPayloadFallsBackToTextarea(t *testing.T) {
	form := url.Values{}
	form.Set("backup_json", `{"from":"textarea"}`)

	req := httptest.NewRequest("POST", "/admin/import", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	payload, err := readBackupPayload(req)
	if err != nil {
		t.Fatalf("readBackupPayload() error = %v", err)
	}
	if got := string(payload); got != `{"from":"textarea"}` {
		t.Fatalf("readBackupPayload() = %q, want textarea payload", got)
	}
}
