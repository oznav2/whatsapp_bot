package messagehandler

import (
	"testing"

	waProto "go.mau.fi/whatsmeow/proto/waE2E"
)

func TestIsAudioMessage(t *testing.T) {
	// Test: audio message
	audioMsg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL: stringPtr("https://example.com/audio.ogg"),
		},
	}

	if !isAudioMessage(audioMsg) {
		t.Error("expected isAudioMessage to return true for audio message")
	}

	// Test: text message
	textMsg := &waProto.Message{
		Conversation: stringPtr("Hello"),
	}

	if isAudioMessage(textMsg) {
		t.Error("expected isAudioMessage to return false for text message")
	}
}

func TestIsPTTVoiceNote(t *testing.T) {
	// Test: PTT voice note
	pttMsg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL: stringPtr("https://example.com/voice.ogg"),
			PTT: boolPtr(true),
		},
	}

	if !isPTTVoiceNote(pttMsg) {
		t.Error("expected isPTTVoiceNote to return true for PTT message")
	}

	// Test: regular audio file
	audioMsg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL: stringPtr("https://example.com/audio.mp3"),
			PTT: boolPtr(false),
		},
	}

	if isPTTVoiceNote(audioMsg) {
		t.Error("expected isPTTVoiceNote to return false for non-PTT audio")
	}

	// Test: PTT not set (defaults to false)
	audioMsgNoPTT := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL: stringPtr("https://example.com/audio.mp3"),
		},
	}

	if isPTTVoiceNote(audioMsgNoPTT) {
		t.Error("expected isPTTVoiceNote to return false when PTT not set")
	}
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
