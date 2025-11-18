package transcription

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	"go.mau.fi/whatsmeow/types"
)

func TestTDisableCommand_Execute(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	stateManager := transcription.NewStateManager(db)

	// Enable transcription first
	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}
	stateManager.Enable(context.Background(), chatJID, "he")

	handler := &mockHandler{
		stateManager: stateManager,
	}

	cmd := NewTDisableCommand()

	ctx := &framework.Context{
		Context:     context.Background(),
		MessageInfo: types.MessageInfo{},
		Command:     "tdisable",
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

	// Verify transcription was disabled
	enabled, err := stateManager.IsEnabled(context.Background(), chatJID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("expected transcription to be disabled")
	}
}
