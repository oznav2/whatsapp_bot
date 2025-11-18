package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	transcriptionhandlers "github.com/asparkoffire/whatsapp-livetranslate-go/internal/handlers/transcription"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

// TestTranscriptionEndToEnd tests the complete transcription flow
func TestTranscriptionEndToEnd(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Create mock transcription server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a multipart request
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Get language from form
		language := r.FormValue("language")

		// Determine model from URL path
		model := "ivrit-ct2"
		if r.URL.Path == "/api/transcribe/whisper" {
			model = "whisper-v3-turbo"
		} else if r.URL.Path == "/api/transcribe/deepgram" {
			model = "deepgram"
		}

		// Verify file was uploaded
		_, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing audio file", http.StatusBadRequest)
			return
		}

		// Return mock transcription response
		response := transcription.TranscriptionResponse{
			Success:          true,
			Status:           "completed",
			Model:            model,
			Language:         language,
			Text:             "This is a test transcription",
			DetectedLanguage: language,
			Timings: transcription.Timings{
				DownloadMs:   100,
				TranscribeMs: 1400,
				TotalMs:      1500,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Initialize state manager and service
	stateManager := transcription.NewStateManager(db)
	transcriptionSvc := transcription.NewService(mockServer.URL)

	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}

	// Test 1: Transcription disabled by default
	enabled, err := stateManager.IsEnabled(context.Background(), chatJID)
	if err != nil {
		t.Fatalf("failed to check enabled state: %v", err)
	}
	if enabled {
		t.Error("expected transcription to be disabled by default")
	}

	// Test 2: Enable transcription
	err = stateManager.Enable(context.Background(), chatJID, "he")
	if err != nil {
		t.Fatalf("failed to enable transcription: %v", err)
	}

	enabled, err = stateManager.IsEnabled(context.Background(), chatJID)
	if err != nil {
		t.Fatalf("failed to check enabled state: %v", err)
	}
	if !enabled {
		t.Error("expected transcription to be enabled")
	}

	// Test 3: Get language setting
	language, err := stateManager.GetLanguage(context.Background(), chatJID)
	if err != nil {
		t.Fatalf("failed to get language: %v", err)
	}
	if language != "he" {
		t.Errorf("expected language 'he', got '%s'", language)
	}

	// Test 4: Create temporary audio file for transcription
	tmpFile, err := os.CreateTemp("", "test_audio_*.ogg")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write some dummy data
	tmpFile.Write([]byte("dummy audio data"))
	tmpFile.Close()

	// Test 5: Transcribe audio
	result, err := transcriptionSvc.TranscribeAudio(context.Background(), tmpFile.Name(), "he", "ivrit-ct2")
	if err != nil {
		t.Fatalf("failed to transcribe audio: %v", err)
	}

	if !result.Success {
		t.Error("expected successful transcription")
	}
	if result.Text != "This is a test transcription" {
		t.Errorf("expected text 'This is a test transcription', got '%s'", result.Text)
	}
	if result.Language != "he" {
		t.Errorf("expected language 'he', got '%s'", result.Language)
	}
	if result.Model != "ivrit-ct2" {
		t.Errorf("expected model 'ivrit-ct2', got '%s'", result.Model)
	}

	// Test 6: Disable transcription
	err = stateManager.Disable(context.Background(), chatJID)
	if err != nil {
		t.Fatalf("failed to disable transcription: %v", err)
	}

	enabled, err = stateManager.IsEnabled(context.Background(), chatJID)
	if err != nil {
		t.Fatalf("failed to check enabled state: %v", err)
	}
	if enabled {
		t.Error("expected transcription to be disabled")
	}
}

// TestCommandExecution tests the tenable and tdisable commands
func TestCommandExecution(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	stateManager := transcription.NewStateManager(db)
	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}

	// Mock handler
	mockHandler := &mockHandlerInterface{
		stateManager: stateManager,
		responses:    []string{},
	}

	// Test TEnableCommand
	tenableCmd := transcriptionhandlers.NewTEnableCommand()

	ctx := &framework.Context{
		Context:     context.Background(),
		Message:     &waProto.Message{},
		MessageInfo: types.MessageInfo{},
		Command:     "tenable",
		Args:        []string{"en"},
		RawArgs:     "en",
		Handler:     mockHandler,
	}
	ctx.MessageInfo.Chat = chatJID

	err = tenableCmd.Execute(ctx)
	if err != nil {
		t.Fatalf("tenable command failed: %v", err)
	}

	// Verify transcription was enabled
	enabled, err := stateManager.IsEnabled(context.Background(), chatJID)
	if err != nil {
		t.Fatalf("failed to check enabled state: %v", err)
	}
	if !enabled {
		t.Error("expected transcription to be enabled after tenable command")
	}

	// Verify language was set
	language, err := stateManager.GetLanguage(context.Background(), chatJID)
	if err != nil {
		t.Fatalf("failed to get language: %v", err)
	}
	if language != "en" {
		t.Errorf("expected language 'en', got '%s'", language)
	}

	// Test TDisableCommand
	tdisableCmd := transcriptionhandlers.NewTDisableCommand()

	ctx.Command = "tdisable"
	ctx.Args = []string{}
	ctx.RawArgs = ""

	err = tdisableCmd.Execute(ctx)
	if err != nil {
		t.Fatalf("tdisable command failed: %v", err)
	}

	// Verify transcription was disabled
	enabled, err = stateManager.IsEnabled(context.Background(), chatJID)
	if err != nil {
		t.Fatalf("failed to check enabled state: %v", err)
	}
	if enabled {
		t.Error("expected transcription to be disabled after tdisable command")
	}
}

// TestTranscriptionServiceErrors tests error handling
func TestTranscriptionServiceErrors(t *testing.T) {
	// Test 1: Service returns error response
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "internal server error",
		})
	}))
	defer errorServer.Close()

	svc := transcription.NewService(errorServer.URL)

	tmpFile, _ := os.CreateTemp("", "test_audio_*.ogg")
	tmpFile.Write([]byte("dummy"))
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err := svc.TranscribeAudio(context.Background(), tmpFile.Name(), "he", "ivrit-ct2")
	if err == nil {
		t.Error("expected error when service returns 500")
	}

	// Test 2: Service timeout
	timeoutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Longer than typical timeout
	}))
	defer timeoutServer.Close()

	svcTimeout := transcription.NewService(timeoutServer.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = svcTimeout.TranscribeAudio(ctx, tmpFile.Name(), "he", "ivrit-ct2")
	if err == nil {
		t.Error("expected timeout error")
	}

	// Test 3: Invalid file path
	_, err = svc.TranscribeAudio(context.Background(), "/nonexistent/file.ogg", "he", "ivrit-ct2")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// TestStatePersistence tests that settings persist across multiple operations
func TestStatePersistence(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	stateManager := transcription.NewStateManager(db)
	chat1 := types.JID{User: "1111111111", Server: "s.whatsapp.net"}
	chat2 := types.JID{User: "2222222222", Server: "s.whatsapp.net"}
	ctx := context.Background()

	// Enable for chat1 with Hebrew
	err = stateManager.Enable(ctx, chat1, "he")
	if err != nil {
		t.Fatalf("failed to enable for chat1: %v", err)
	}

	// Enable for chat2 with English
	err = stateManager.Enable(ctx, chat2, "en")
	if err != nil {
		t.Fatalf("failed to enable for chat2: %v", err)
	}

	// Verify chat1 settings
	enabled1, _ := stateManager.IsEnabled(ctx, chat1)
	lang1, _ := stateManager.GetLanguage(ctx, chat1)
	if !enabled1 || lang1 != "he" {
		t.Error("chat1 settings not persisted correctly")
	}

	// Verify chat2 settings
	enabled2, _ := stateManager.IsEnabled(ctx, chat2)
	lang2, _ := stateManager.GetLanguage(ctx, chat2)
	if !enabled2 || lang2 != "en" {
		t.Error("chat2 settings not persisted correctly")
	}

	// Disable chat1
	err = stateManager.Disable(ctx, chat1)
	if err != nil {
		t.Fatalf("failed to disable chat1: %v", err)
	}

	// Verify chat1 disabled, chat2 still enabled
	enabled1, _ = stateManager.IsEnabled(ctx, chat1)
	enabled2, _ = stateManager.IsEnabled(ctx, chat2)
	if enabled1 {
		t.Error("chat1 should be disabled")
	}
	if !enabled2 {
		t.Error("chat2 should still be enabled")
	}
}

// mockHandlerInterface implements the handler interface for testing
type mockHandlerInterface struct {
	stateManager *transcription.StateManager
	responses    []string
}

func (m *mockHandlerInterface) SendResponse(msgInfo types.MessageInfo, text string) error {
	m.responses = append(m.responses, text)
	return nil
}

func (m *mockHandlerInterface) GetStateManager() *transcription.StateManager {
	return m.stateManager
}

func (m *mockHandlerInterface) GetTranscriptionService() *transcription.Service {
	return nil
}

// Implement all required HandlerInterface methods
func (m *mockHandlerInterface) SendMedia(msgInfo types.MessageInfo, mediaType framework.MediaType, data []byte, caption string) error {
	return nil
}

func (m *mockHandlerInterface) SendImage(msgInfo types.MessageInfo, upload framework.UploadResponse, caption string) error {
	return nil
}

func (m *mockHandlerInterface) SendVideo(msgInfo types.MessageInfo, upload framework.UploadResponse, caption string) error {
	return nil
}

func (m *mockHandlerInterface) SendDocument(msgInfo types.MessageInfo, upload framework.UploadResponse, caption string) error {
	return nil
}

func (m *mockHandlerInterface) EditMessage(msgInfo types.MessageInfo, newText string) error {
	return nil
}

func (m *mockHandlerInterface) EditMessageWithOriginal(msgInfo types.MessageInfo, newText string, originalMsg *waProto.Message) error {
	return nil
}

func (m *mockHandlerInterface) GetClient() framework.ClientInterface {
	return nil
}

func (m *mockHandlerInterface) GetTranslator() framework.TranslatorInterface {
	return nil
}

func (m *mockHandlerInterface) GetImageGenerator() framework.ImageGeneratorInterface {
	return nil
}

func (m *mockHandlerInterface) GetMemeGenerator() framework.MemeGeneratorInterface {
	return nil
}

func (m *mockHandlerInterface) GetLangDetector() framework.LangDetectorInterface {
	return nil
}
