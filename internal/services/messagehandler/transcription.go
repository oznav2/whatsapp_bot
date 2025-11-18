package messagehandler

import (
	"context"
	"fmt"
	"os"

	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
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

	// Send initial "Transcribing..." status message
	senderJID := msgInfo.Chat
	if msgInfo.Chat.Server == "g.us" {
		// In groups, use appropriate participant JID
		senderJID = types.NewJID(msgInfo.Chat.User, "s.whatsapp.net")
	}

	// Create the initial "Transcribing..." message as a reply to the audio
	initialMsg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String("🎤 Transcribing..."),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:    proto.String(msgInfo.ID),
				Participant: proto.String(senderJID.String()),
			},
		},
	}

	resp, err := h.client.SendMessage(ctx, msgInfo.Chat, initialMsg)
	if err != nil {
		fmt.Printf("Failed to send 'Transcribing...' message: %v\n", err)
		return nil // Don't fail the whole operation
	}

	fmt.Printf("Sent 'Transcribing...' status message (ID: %s)\n", resp.ID)

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

	// DEBUG: Log the full result
	fmt.Printf("[DEBUG] Transcription result for chat %s:\n", msgInfo.Chat.String())
	fmt.Printf("[DEBUG]   Success: %v\n", result.Success)
	fmt.Printf("[DEBUG]   Text length: %d\n", len(result.Text))
	fmt.Printf("[DEBUG]   Text: %q\n", result.Text)
	fmt.Printf("[DEBUG]   Model: %s\n", result.Model)
	fmt.Printf("[DEBUG]   Language: %s\n", result.Language)
	fmt.Printf("[DEBUG]   DetectedLanguage: %s\n", result.DetectedLanguage)

	// Build the final transcription message
	response := fmt.Sprintf("🎤 *Transcription:*\n\n%s", result.Text)
	if result.DetectedLanguage != "" && result.DetectedLanguage != language {
		response += fmt.Sprintf("\n\n🌐 Detected language: %s", result.DetectedLanguage)
	}

	// DEBUG: Log the response being sent
	fmt.Printf("[DEBUG] Editing status message with transcription (length: %d)\n", len(response))

	// Edit the "Transcribing..." message with the actual transcription
	// Create the updated message with the same context info
	updatedMsg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(response),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:    proto.String(msgInfo.ID),
				Participant: proto.String(senderJID.String()),
			},
		},
	}

	editMsg := h.client.BuildEdit(msgInfo.Chat, resp.ID, updatedMsg)
	_, err = h.client.SendMessage(ctx, msgInfo.Chat, editMsg)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to edit transcription message: %v\n", err)
		return fmt.Errorf("failed to edit transcription message: %w", err)
	}

	// DEBUG: Confirm send completed
	fmt.Printf("[DEBUG] Transcription message edited successfully for chat %s\n", msgInfo.Chat.String())

	return nil
}
