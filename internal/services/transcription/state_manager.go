package transcription

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/types"
)

type StateManager struct {
	db *sql.DB
}

func NewStateManager(db *sql.DB) *StateManager {
	manager := &StateManager{db: db}
	manager.initSchema()
	return manager
}

func (sm *StateManager) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS transcription_settings (
		chat_jid TEXT PRIMARY KEY,
		enabled INTEGER NOT NULL DEFAULT 0,
		language TEXT DEFAULT 'he',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_transcription_enabled
	ON transcription_settings(enabled);
	`

	_, err := sm.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

func (sm *StateManager) IsEnabled(ctx context.Context, chatJID types.JID) (bool, error) {
	var enabled int
	query := "SELECT enabled FROM transcription_settings WHERE chat_jid = ?"
	err := sm.db.QueryRowContext(ctx, query, chatJID.String()).Scan(&enabled)

	if err == sql.ErrNoRows {
		return false, nil // Default to disabled
	}
	if err != nil {
		return false, fmt.Errorf("failed to check if enabled: %w", err)
	}

	return enabled == 1, nil
}

func (sm *StateManager) Enable(ctx context.Context, chatJID types.JID, language string) error {
	query := `
	INSERT INTO transcription_settings (chat_jid, enabled, language, updated_at)
	VALUES (?, 1, ?, ?)
	ON CONFLICT(chat_jid) DO UPDATE SET
		enabled = 1,
		language = excluded.language,
		updated_at = excluded.updated_at
	`

	_, err := sm.db.ExecContext(ctx, query, chatJID.String(), language, time.Now())
	if err != nil {
		return fmt.Errorf("failed to enable transcription: %w", err)
	}
	return nil
}

func (sm *StateManager) Disable(ctx context.Context, chatJID types.JID) error {
	query := `
	INSERT INTO transcription_settings (chat_jid, enabled, updated_at)
	VALUES (?, 0, ?)
	ON CONFLICT(chat_jid) DO UPDATE SET
		enabled = 0,
		updated_at = excluded.updated_at
	`

	_, err := sm.db.ExecContext(ctx, query, chatJID.String(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to disable transcription: %w", err)
	}
	return nil
}

func (sm *StateManager) GetLanguage(ctx context.Context, chatJID types.JID) (string, error) {
	var language string
	query := "SELECT language FROM transcription_settings WHERE chat_jid = ?"
	err := sm.db.QueryRowContext(ctx, query, chatJID.String()).Scan(&language)

	if err == sql.ErrNoRows {
		return "he", nil // Default to Hebrew
	}
	if err != nil {
		return "", fmt.Errorf("failed to get language: %w", err)
	}

	return language, nil
}
