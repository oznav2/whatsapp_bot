package transcription

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestService_TranscribeAudio(t *testing.T) {
	// Create mock transcription server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/transcribe/whisper-ivrit" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("no file in form: %v", err)
		}
		defer file.Close()

		language := r.FormValue("language")
		if language != "he" {
			t.Errorf("expected language 'he', got '%s'", language)
		}

		// Return mock response
		response := TranscriptionResponse{
			Success:          true,
			Status:           "ok",
			Model:            "ivrit-ct2",
			Language:         "he",
			Text:             "שלום עולם",
			DetectedLanguage: "he",
			Timings: Timings{
				DownloadMs:   100,
				TranscribeMs: 2000,
				TotalMs:      2100,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create test audio file
	tmpFile, err := os.CreateTemp("", "test_audio_*.ogg")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write some dummy data
	tmpFile.Write([]byte("dummy audio data"))

	service := NewService(server.URL)
	ctx := context.Background()

	result, err := service.TranscribeAudio(ctx, tmpFile.Name(), "he", "ivrit-ct2")
	if err != nil {
		t.Fatalf("TranscribeAudio failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success = true")
	}
	if result.Text != "שלום עולם" {
		t.Errorf("expected text 'שלום עולם', got '%s'", result.Text)
	}
	if result.Language != "he" {
		t.Errorf("expected language 'he', got '%s'", result.Language)
	}
}

func TestService_TranscribeAudio_Error(t *testing.T) {
	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Transcription service unavailable",
		})
	}))
	defer server.Close()

	tmpFile, err := os.CreateTemp("", "test_audio_*.ogg")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	tmpFile.Write([]byte("dummy audio data"))

	service := NewService(server.URL)
	ctx := context.Background()

	_, err = service.TranscribeAudio(ctx, tmpFile.Name(), "he", "ivrit-ct2")
	if err == nil {
		t.Error("expected error, got nil")
	}
}
