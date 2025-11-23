package transcription

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// GetVideoMetadata fetches metadata about a video from the transcription service
func (s *Service) GetVideoMetadata(ctx context.Context, videoURL string) (*VideoMetadata, error) {
	// Construct request URL
	apiURL := s.baseURL + "/api/video-info"

	// Create request body
	requestBody := map[string]string{
		"url": videoURL,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		// If video-info endpoint doesn't exist, return basic metadata
		return &VideoMetadata{
			Title:           "",
			DurationSeconds: 0,
		}, nil
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		// If endpoint fails, return basic metadata instead of error
		return &VideoMetadata{
			Title:           "",
			DurationSeconds: 0,
		}, nil
	}

	// Parse response
	var result VideoInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &VideoMetadata{
			Title:           "",
			DurationSeconds: 0,
		}, nil
	}

	if !result.Success {
		return &VideoMetadata{
			Title:           "",
			DurationSeconds: 0,
		}, nil
	}

	return &result.Metadata, nil
}

// QuickLanguageDetection performs fast language detection using first 60 seconds
func (s *Service) QuickLanguageDetection(ctx context.Context, videoURL string) (string, error) {
	// Try Hebrew first (ivrit-ct2 model) since Deepgram doesn't support Hebrew well
	hebrewRequest := WSTranscriptionRequest{
		URL:         videoURL,
		Language:    "he",
		Model:       "ivrit-ct2",
		CaptureMode: "first60",
	}

	result, err := s.TranscribeViaWebSocket(ctx, hebrewRequest, nil)
	if err == nil && result != nil && strings.TrimSpace(result.Text) != "" {
		// If Hebrew transcription succeeded and returned text, it's Hebrew
		return "he", nil
	}

	// If Hebrew failed, try Deepgram with auto-detect for other languages
	deepgramRequest := WSTranscriptionRequest{
		URL:         videoURL,
		CaptureMode: "first60",
		Model:       "deepgram",
	}

	var detectedLang string
	deepgramCallback := func(msg WSTranscriptionMessage) {
		if msg.Type == "complete" && msg.DetectedLanguage != "" {
			detectedLang = msg.DetectedLanguage
		}
	}

	result, err = s.TranscribeViaWebSocket(ctx, deepgramRequest, deepgramCallback)
	if err != nil {
		// If both failed, default to English
		return "en", nil
	}

	if detectedLang == "" || detectedLang == "unknown" {
		return "en", nil
	}

	return detectedLang, nil
}

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
		case "transcription_chunk":
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
