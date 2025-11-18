package messagehandler

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

func TestShouldTranscribe(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	stateManager := transcription.NewStateManager(db)
	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}

	// Test: disabled by default
	audioMsg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL: stringPtr("https://example.com/audio.ogg"),
		},
	}

	should := shouldTranscribe(audioMsg, chatJID, stateManager)
	if should {
		t.Error("expected shouldTranscribe = false when disabled")
	}

	// Enable transcription
	stateManager.Enable(context.Background(), chatJID, "he")

	should = shouldTranscribe(audioMsg, chatJID, stateManager)
	if !should {
		t.Error("expected shouldTranscribe = true when enabled")
	}

	// Test: non-audio message
	textMsg := &waProto.Message{
		Conversation: stringPtr("Hello"),
	}

	should = shouldTranscribe(textMsg, chatJID, stateManager)
	if should {
		t.Error("expected shouldTranscribe = false for non-audio message")
	}
}
