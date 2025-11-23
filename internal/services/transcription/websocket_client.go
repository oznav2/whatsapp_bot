package transcription

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// TranscribeViaWebSocket performs transcription using WebSocket for real-time progress
func (s *Service) TranscribeViaWebSocket(ctx context.Context, request WSTranscriptionRequest, progressCallback func(WSTranscriptionMessage)) (*TranscriptionResponse, error) {
	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(s.baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL += "/ws/transcribe"

	// Parse URL
	u, err := url.Parse(wsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WebSocket URL: %w", err)
	}

	// Create WebSocket connection
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()

	// Send transcription request
	if err := conn.WriteJSON(request); err != nil {
		return nil, fmt.Errorf("failed to send WebSocket request: %w", err)
	}

	// Collect transcription chunks
	var transcriptionChunks []string
	var detectedLanguage string
	var finalText string

	// Read messages from WebSocket
	for {
		var msg WSTranscriptionMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				break
			}
			return nil, fmt.Errorf("failed to read WebSocket message: %w", err)
		}

		// Call progress callback if provided
		if progressCallback != nil {
			progressCallback(msg)
		}

		// Handle message types
		switch msg.Type {
		case "transcription_chunk", "transcription":
			// VibeGram sends "transcription" for Deepgram (full text)
			// and "transcription_chunk" for Whisper models (incremental)
			if msg.Text != "" {
				transcriptionChunks = append(transcriptionChunks, msg.Text)
			}
		case "complete":
			detectedLanguage = msg.DetectedLanguage
			if msg.FinalText != "" {
				finalText = msg.FinalText
			} else {
				// Combine all chunks
				finalText = strings.Join(transcriptionChunks, " ")
			}
			// Close connection gracefully
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			goto done
		case "error":
			return nil, fmt.Errorf("transcription error: %s", msg.Error)
		}
	}

done:
	// Return transcription response
	result := &TranscriptionResponse{
		Success:          true,
		Text:             strings.TrimSpace(finalText),
		DetectedLanguage: detectedLanguage,
		Language:         detectedLanguage,
		Status:           "completed",
	}

	return result, nil
}
