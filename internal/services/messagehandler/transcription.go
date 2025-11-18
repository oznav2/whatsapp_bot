package messagehandler

import (
	"context"
	"fmt"
	"os"

	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

func shouldTranscribe(msg *waProto.Message, chatJID types.JID, stateManager *transcription.StateManager) bool {
	// Check if it's an audio message
	if msg.GetAudioMessage() == nil {
		return false
	}

	// Check if transcription is enabled for this chat
	enabled, err := stateManager.IsEnabled(context.Background(), chatJID)
	if err != nil || !enabled {
		return false
	}

	return true
}

func (h *WhatsMeowEventHandler) handleAudioTranscription(msg *waProto.Message, msgInfo types.MessageInfo) error {
	ctx := context.Background()

	// Download audio using whatsmeow's client.Download() (PROVEN PATTERN)
	audioMsg := msg.GetAudioMessage()
	audioData, err := h.client.Download(ctx, audioMsg)
	if err != nil {
		return fmt.Errorf("failed to download audio: %w", err)
	}

	// Save to temp file (PROVEN PATTERN from all reference repos)
	tmpFile, err := os.CreateTemp("", "whatsapp_audio_*.ogg")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = tmpFile.Write(audioData)
	if err != nil {
		return fmt.Errorf("failed to write audio data: %w", err)
	}
	tmpFile.Close() // Close before reading

	// Get language preference
	language, err := h.transcriptionState.GetLanguage(ctx, msgInfo.Chat)
	if err != nil {
		language = "he" // Default to Hebrew
	}

	// Determine model based on language
	model := "ivrit-ct2" // Default for Hebrew
	if language != "he" {
		model = "whisper-v3-turbo"
	}

	// Transcribe
	result, err := h.transcriptionSvc.TranscribeAudio(ctx, tmpFile.Name(), language, model)
	if err != nil {
		// Silent fail - don't spam user if transcription service is down (PROVEN PATTERN)
		fmt.Printf("Transcription failed for chat %s: %v\n", msgInfo.Chat.String(), err)
		return nil
	}

	// Send transcription as reply
	adapter := NewHandlerAdapter(h)
	response := fmt.Sprintf("🎤 *Transcription:*\n\n%s", result.Text)
	if result.DetectedLanguage != "" && result.DetectedLanguage != language {
		response += fmt.Sprintf("\n\n🌐 Detected language: %s", result.DetectedLanguage)
	}

	adapter.SendResponse(msgInfo, response)
	return nil
}
