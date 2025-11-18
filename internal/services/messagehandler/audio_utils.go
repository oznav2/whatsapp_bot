package messagehandler

import (
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
)

// isAudioMessage checks if the message contains audio
func isAudioMessage(msg *waProto.Message) bool {
	return msg.GetAudioMessage() != nil
}

// isPTTVoiceNote checks if the audio message is a push-to-talk voice note
// This distinguishes voice messages from audio file attachments
func isPTTVoiceNote(msg *waProto.Message) bool {
	audioMsg := msg.GetAudioMessage()
	if audioMsg == nil {
		return false
	}
	return audioMsg.GetPTT()
}
