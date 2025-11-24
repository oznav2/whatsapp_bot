package messagehandler

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func shouldTranscribe(msg *waProto.Message, chatJID types.JID, stateManager *transcription.StateManager) bool {
	// Check if it's an audio or video message
	if msg.GetAudioMessage() == nil && msg.GetVideoMessage() == nil {
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
			Text: proto.String("🎤 מתמלל הודעה..."),
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
	response := fmt.Sprintf("🎤 *תמלול הודעה קולית:*\n\n%s", result.Text)
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

func (h *WhatsMeowEventHandler) handleVideoTranscription(msg *waProto.Message, msgInfo types.MessageInfo) error {
	ctx := context.Background()

	// Send initial "Downloading video..." status message
	senderJID := msgInfo.Chat
	if msgInfo.Chat.Server == "g.us" {
		senderJID = types.NewJID(msgInfo.Chat.User, "s.whatsapp.net")
	}

	initialMsg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String("📹 מוריד וידאו..."),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:    proto.String(msgInfo.ID),
				Participant: proto.String(senderJID.String()),
			},
		},
	}

	resp, err := h.client.SendMessage(ctx, msgInfo.Chat, initialMsg)
	if err != nil {
		fmt.Printf("Failed to send 'Downloading video...' message: %v\n", err)
		return nil
	}

	fmt.Printf("Sent 'Downloading video...' status message (ID: %s)\n", resp.ID)

	// Download video using whatsmeow's client.Download()
	videoMsg := msg.GetVideoMessage()
	videoData, err := h.client.Download(ctx, videoMsg)
	if err != nil {
		h.editStatusMessage(ctx, msgInfo.Chat, resp.ID, msgInfo.ID, senderJID, "❌ שגיאה בהורדת הווידאו")
		return fmt.Errorf("failed to download video: %w", err)
	}

	// Save to temp file
	tmpFile, err := os.CreateTemp("", "whatsapp_video_*.mp4")
	if err != nil {
		h.editStatusMessage(ctx, msgInfo.Chat, resp.ID, msgInfo.ID, senderJID, "❌ שגיאה בשמירת הווידאו")
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = tmpFile.Write(videoData)
	if err != nil {
		h.editStatusMessage(ctx, msgInfo.Chat, resp.ID, msgInfo.ID, senderJID, "❌ שגיאה בשמירת הווידאו")
		return fmt.Errorf("failed to write video data: %w", err)
	}
	tmpFile.Close()

	// Upload video to transcription service
	h.editStatusMessage(ctx, msgInfo.Chat, resp.ID, msgInfo.ID, senderJID, "📤 מעלה וידאו לשירות תמלול...")
	uploadResp, err := h.transcriptionSvc.UploadAudio(ctx, tmpFile.Name())
	if err != nil {
		h.editStatusMessage(ctx, msgInfo.Chat, resp.ID, msgInfo.ID, senderJID, "❌ שגיאה בהעלאת הווידאו")
		fmt.Printf("Failed to upload video: %v\n", err)
		return nil
	}

	// Use the file ID directly for uploaded files
	uploadedFileID := uploadResp.FileID

	// Get video metadata (duration, etc.)
	h.editStatusMessage(ctx, msgInfo.Chat, resp.ID, msgInfo.ID, senderJID, "📊 מקבל מידע על הווידאו...")

	// For WhatsApp videos, we don't have full metadata, so create basic metadata
	videoLengthSeconds := int(videoMsg.GetSeconds())
	videoDuration := transcription.FormatDuration(videoLengthSeconds)

	// Get language preference
	language, err := h.transcriptionState.GetLanguage(ctx, msgInfo.Chat)
	if err != nil {
		language = "he" // Default to Hebrew
	}

	// Quick language detection from first 60 seconds using Hebrew transcription first
	h.editStatusMessage(ctx, msgInfo.Chat, resp.ID, msgInfo.ID, senderJID, "🔍 מזהה שפה...")

	quickRequest := transcription.WSTranscriptionRequest{
		UploadFileID: uploadedFileID,
		Language:     "he",
		Model:        "ivrit-ct2",
		CaptureMode:  "first60",
	}

	quickResult, _ := h.transcriptionSvc.TranscribeViaWebSocket(ctx, quickRequest, nil)

	// Use existing language detector on transcribed text
	detectedLangCode := language // Default to configured language
	languageName := "Hebrew"
	model := "ivrit-ct2"

	if quickResult != nil && strings.TrimSpace(quickResult.Text) != "" {
		// Use the bot's existing language detection on the transcribed text
		detectedLang, ok := h.detector.DetectLanguage(quickResult.Text)
		if ok {
			detectedLangCode = strings.ToLower(detectedLang.IsoCode639_1().String())
			// If not Hebrew, use Whisper multilingual
			if detectedLangCode != "he" && detectedLangCode != "iw" {
				model = "whisper-v3-turbo"
				languageName = detectedLangCode
			}
		}
	} else {
		// Hebrew transcription failed, try Whisper multilingual
		model = "whisper-v3-turbo"
		detectedLangCode = "en"
		languageName = "English"
	}

	// Show verbose progress message
	verboseMsg := fmt.Sprintf("🎬 מתמלל וידאו...\n⏱️ משך: %s\n🔤 שפה מזוהה: %s", videoDuration, languageName)
	h.editStatusMessage(ctx, msgInfo.Chat, resp.ID, msgInfo.ID, senderJID, verboseMsg)

	// Full transcription with progress tracking
	request := transcription.WSTranscriptionRequest{
		UploadFileID: uploadedFileID,
		Language:     detectedLangCode,
		Model:        model,
		CaptureMode:  "full",
	}

	lastPercent := 0.0
	progressCallback := func(wsMsg transcription.WSTranscriptionMessage) {
		if wsMsg.Type == "download_progress" {
			// Update only every 25%
			if wsMsg.Percent-lastPercent >= 25.0 || wsMsg.Percent >= 99.0 {
				lastPercent = wsMsg.Percent
				progressMsg := fmt.Sprintf("🎬 מתמלל וידאו... (הורדה: %.0f%%)\n⏱️ משך: %s\n🔤 שפה מזוהה: %s",
					wsMsg.Percent, videoDuration, languageName)
				h.editStatusMessage(ctx, msgInfo.Chat, resp.ID, msgInfo.ID, senderJID, progressMsg)
			}
		}
		// DO NOT show transcription chunks - collect silently
	}

	result, err := h.transcriptionSvc.TranscribeViaWebSocket(ctx, request, progressCallback)
	if err != nil {
		h.editStatusMessage(ctx, msgInfo.Chat, resp.ID, msgInfo.ID, senderJID, "❌ שגיאה בתמלול הווידאו")
		fmt.Printf("Transcription failed: %v\n", err)
		return nil
	}

	// Present final complete transcription
	finalResponse := fmt.Sprintf("🎬 *תמלול הושלם*\n\n⏱️ משך: %s\n🔤 שפה: %s\n\n📝 *תמלול:*\n\n%s",
		videoDuration, languageName, result.Text)

	h.editStatusMessage(ctx, msgInfo.Chat, resp.ID, msgInfo.ID, senderJID, finalResponse)

	fmt.Printf("Video transcription completed successfully for chat %s\n", msgInfo.Chat.String())
	return nil
}

// Helper function to edit status messages
func (h *WhatsMeowEventHandler) editStatusMessage(ctx context.Context, chatJID types.JID, messageID string, replyToID string, senderJID types.JID, text string) {
	updatedMsg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:    proto.String(replyToID),
				Participant: proto.String(senderJID.String()),
			},
		},
	}

	editMsg := h.client.BuildEdit(chatJID, messageID, updatedMsg)
	_, err := h.client.SendMessage(ctx, chatJID, editMsg)
	if err != nil {
		fmt.Printf("Failed to edit status message: %v\n", err)
	}
}
