package transcription

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/types"
)

func TestStateManager_IsEnabled(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	manager := NewStateManager(db)
	ctx := context.Background()

	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}

	// Test: new chat should default to disabled
	enabled, err := manager.IsEnabled(ctx, chatJID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("expected transcription to be disabled by default")
	}
}

func TestStateManager_Enable(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	manager := NewStateManager(db)
	ctx := context.Background()

	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}

	// Enable transcription
	err = manager.Enable(ctx, chatJID, "he")
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	// Verify it's enabled
	enabled, err := manager.IsEnabled(ctx, chatJID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if !enabled {
		t.Error("expected transcription to be enabled")
	}

	// Verify language
	lang, err := manager.GetLanguage(ctx, chatJID)
	if err != nil {
		t.Fatalf("GetLanguage failed: %v", err)
	}
	if lang != "he" {
		t.Errorf("expected language 'he', got '%s'", lang)
	}
}

func TestStateManager_Disable(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	manager := NewStateManager(db)
	ctx := context.Background()

	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}

	// Enable first
	_ = manager.Enable(ctx, chatJID, "he")

	// Disable
	err = manager.Disable(ctx, chatJID)
	if err != nil {
		t.Fatalf("Disable failed: %v", err)
	}

	// Verify it's disabled
	enabled, err := manager.IsEnabled(ctx, chatJID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("expected transcription to be disabled")
	}
}
