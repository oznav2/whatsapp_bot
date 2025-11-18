package transcription

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

type mockHandler struct {
	lastResponse string
	stateManager *transcription.StateManager
}

func (m *mockHandler) SendResponse(msgInfo types.MessageInfo, text string) error {
	m.lastResponse = text
	return nil
}

func (m *mockHandler) GetStateManager() *transcription.StateManager {
	return m.stateManager
}

// Implement other required HandlerInterface methods as no-ops
func (m *mockHandler) SendMedia(msgInfo types.MessageInfo, mediaType framework.MediaType, data []byte, caption string) error {
	return nil
}
func (m *mockHandler) SendImage(msgInfo types.MessageInfo, upload framework.UploadResponse, caption string) error {
	return nil
}
func (m *mockHandler) SendVideo(msgInfo types.MessageInfo, upload framework.UploadResponse, caption string) error {
	return nil
}
func (m *mockHandler) SendDocument(msgInfo types.MessageInfo, upload framework.UploadResponse, caption string) error {
	return nil
}
func (m *mockHandler) EditMessage(msgInfo types.MessageInfo, newText string) error { return nil }
func (m *mockHandler) EditMessageWithOriginal(msgInfo types.MessageInfo, newText string, originalMsg *waProto.Message) error {
	return nil
}
func (m *mockHandler) GetClient() framework.ClientInterface           { return nil }
func (m *mockHandler) GetTranslator() framework.TranslatorInterface   { return nil }
func (m *mockHandler) GetImageGenerator() framework.ImageGeneratorInterface {
	return nil
}
func (m *mockHandler) GetMemeGenerator() framework.MemeGeneratorInterface { return nil }
func (m *mockHandler) GetLangDetector() framework.LangDetectorInterface   { return nil }

func TestTEnableCommand_Execute(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	stateManager := transcription.NewStateManager(db)

	handler := &mockHandler{
		stateManager: stateManager,
	}

	cmd := NewTEnableCommand()

	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}
	ctx := &framework.Context{
		Context:     context.Background(),
		MessageInfo: types.MessageInfo{},
		Command:     "tenable",
		Args:        []string{},
		RawArgs:     "",
		Handler:     handler,
	}
	ctx.MessageInfo.Chat = chatJID

	err = cmd.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify response was sent
	if handler.lastResponse == "" {
		t.Error("expected response, got empty string")
	}

	// Verify transcription was enabled
	enabled, err := stateManager.IsEnabled(context.Background(), ctx.MessageInfo.Chat)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if !enabled {
		t.Error("expected transcription to be enabled")
	}
}

func TestTEnableCommand_ExecuteWithLanguage(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	stateManager := transcription.NewStateManager(db)

	handler := &mockHandler{
		stateManager: stateManager,
	}

	cmd := NewTEnableCommand()

	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}
	ctx := &framework.Context{
		Context:     context.Background(),
		MessageInfo: types.MessageInfo{},
		Command:     "tenable",
		Args:        []string{"en"},
		RawArgs:     "en",
		Handler:     handler,
	}
	ctx.MessageInfo.Chat = chatJID

	err = cmd.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify language was set
	lang, err := stateManager.GetLanguage(context.Background(), ctx.MessageInfo.Chat)
	if err != nil {
		t.Fatalf("GetLanguage failed: %v", err)
	}
	if lang != "en" {
		t.Errorf("expected language 'en', got '%s'", lang)
	}
}
