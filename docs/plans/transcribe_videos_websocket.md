# Video Transcription Feature Implementation Plan (WebSocket + Language Detection + Smart UX)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extend the existing audio transcription feature to support video messages and video URLs with automatic language detection, optimal model selection via WebSocket connections, and intelligent UX with video metadata display.

**Architecture:** Use Deepgram container's WebSocket API (`ws://localhost:8009/ws/transcribe`) for real-time transcription. Fetch video metadata FIRST to show users what to expect (title, duration). Implement intelligent language detection by transcribing first 60 seconds, then restart with optimal model (ivrit-ct2 for Hebrew, Deepgram for others). Show only download progress during transcription, present final complete transcription when done.

**Tech Stack:** Go, whatsmeow, gorilla/websocket (already in go.mod), Deepgram WebSocket API, Deepgram REST API (/api/video-info for metadata), SQLite3 (existing)

---

## 🚨 CRITICAL IMPLEMENTATION GUIDELINES

### ❌ ABSOLUTE PROHIBITIONS

**Throughout the ENTIRE implementation process, you are EXPLICITLY FORBIDDEN from:**

- ❌ Running `docker build`
- ❌ Running `docker-compose build`
- ❌ Running `docker run`
- ❌ Testing in Docker containers
- ❌ Suggesting Docker builds for verification
- ❌ Creating test files (no `*_test.go` files)
- ❌ Running `go test`
- ❌ Adding yt-dlp to the bot (Deepgram already has it)
- ❌ Adding ffmpeg to the bot (Deepgram already has it)
- ❌ Downloading videos in the bot for URLs (pass URL to Deepgram directly)
- ❌ Extracting audio from videos in the bot (Deepgram does this automatically)

**WHY:**
- User will manually test all functionality
- Docker builds are time-consuming and unnecessary during development
- User requested explicit NO TEST FILES policy
- **Deepgram container already has yt-dlp and ffmpeg - NO REDUNDANCY**
- User will handle Docker build ONCE after ALL implementation phases are complete

---

## Architecture Deep Dive

### Enhanced User Experience Flow

**Smart UX Requirements:**
1. **Fetch video metadata FIRST** - Show user what to expect
   - Video title
   - Video duration (formatted nicely)
2. **Verbose progress message** - Tell user:
   - What video is being transcribed
   - How long it is
   - What language was detected
3. **NO streaming chunks** - Don't show partial transcription text
   - Only show download progress (%)
   - Don't update with partial text chunks
4. **Final transcription only** - Present complete result when done

**Example Message Flow:**
```
1. "🎬 מתחיל תמלול...\n\n🔍 מזהה שפת הסרטון..."

2. "🎬 Transcribing now "How to Cook Pasta" length: 12min and 13 seconds and Video Language detected: Hebrew"

3. "🎬 Transcribing now "How to Cook Pasta"...\n\n📥 Downloading: 25%"

4. "🎬 Transcribing now "How to Cook Pasta"...\n\n📥 Downloading: 50%"

5. "🎬 Transcribing now "How to Cook Pasta"...\n\n🎙️ Transcribing audio..."

6. "🎬 *תמלול הושלם*\n\n📹 סרטון: "How to Cook Pasta"\n⏱️ משך: 12min and 13 seconds\n🔤 שפה: עברית\n\n📝 *תמלול:*\n\n[Full transcription text here]"
```

### Deepgram API Endpoints

**Video Metadata Endpoint:** `POST /api/video-info`

**Request:**
```json
{
  "url": "https://www.youtube.com/watch?v=..."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "title": "How to Cook Pasta",
    "channel": "Cooking Channel",
    "duration_seconds": 733,
    "duration_formatted": "12:13",
    "view_count": 1234567,
    "view_count_formatted": "1.2M",
    "thumbnail": "https://i.ytimg.com/...",
    "is_youtube": true
  }
}
```

**WebSocket Endpoint:** `ws://localhost:8009/ws/transcribe`

**Request Message:**
```json
{
  "url": "https://www.youtube.com/shorts/...",
  "model": "whisper-v3-turbo",
  "language": "he",
  "captureMode": "full",
  "diarization": false
}
```

**Response Messages:**
```json
// Status updates
{"type": "status", "message": "Starting transcription..."}
{"type": "status", "message": "📥 Attempting download with yt-dlp..."}

// Download progress (SHOW THIS)
{"type": "download_progress", "percent": 45.3, "downloaded_mb": 12.34, "speed_mbps": 1.23}

// Transcription chunks (DO NOT SHOW - collect silently)
{"type": "transcription_chunk", "text": "transcribed text...", "chunk_index": 1, "is_final": false}

// Completion (SHOW FINAL RESULT)
{"type": "complete", "message": "Transcription complete", "detected_language": "he"}
```

**Models Available:**
- `whisper-v3-turbo` - Fast multilingual model (good for all languages)
- `ivrit-ct2` - Hebrew-optimized model (best for Hebrew)
- `deepgram` - Deepgram Nova-3 cloud model (best for non-Hebrew)

### Language Detection Strategy

**Simplified Approach:** Use Deepgram's built-in language detection from first transcription.

**Enhanced Flow with Video Metadata:**

**Phase 1: Fetch Video Metadata**
1. POST to `/api/video-info` with video URL
2. Get: title, duration_seconds, duration_formatted
3. Show user: "Found video: "[title]" (duration)"

**Phase 2: Quick Language Detection**
4. Open WebSocket to Deepgram with `whisper-v3-turbo`
5. Request first 60 seconds: `{"captureMode": "first60"}`
6. Wait for completion message with `detected_language`
7. Close WebSocket

**Phase 3: Verbose Progress Message**
8. Format duration nicely (e.g., "12min and 13 seconds")
9. Show user: "Transcribing now "[Title]" length: [duration] and Video Language detected: [language]"
10. Tell user which model will be used (ivrit-ct2 or Deepgram)

**Phase 4: Full Transcription**
11. Open new WebSocket with optimal model
12. Request full video: `{"captureMode": "full"}`
13. Show download progress only (%)
14. Collect transcription chunks silently (don't show partial text)
15. On completion, show final transcription with metadata

**Benefits:**
- ✅ User knows what video is being transcribed
- ✅ User knows how long to wait (based on duration)
- ✅ User knows what language was detected
- ✅ No confusing partial text updates
- ✅ Clean final result presentation

### Current Audio Transcription Flow (Reference)

**Current implementation uses REST API:**
```go
// internal/services/transcription/service.go
uploadResp := service.UploadAudio(ctx, "/tmp/audio.ogg")  // POST /api/upload
result := service.TranscribeAudio(ctx, filePath, "he", "ivrit-ct2")  // POST /api/transcribe/*
```

**New implementation will use WebSocket for URLs:**
```go
// Connect WebSocket
conn := websocket.Dial("ws://localhost:8009/ws/transcribe")
// Send request
conn.WriteJSON({"url": "...", "model": "ivrit-ct2"})
// Read chunks (collect silently, don't display)
for {
    var msg TranscriptionMessage
    conn.ReadJSON(&msg)
    // Collect chunks, show progress only
}
```

---

## Phase 1: WebSocket Client Infrastructure

### Task 1: Create WebSocket Transcription Types

**Files:**
- Create: `internal/services/transcription/websocket_types.go`

**Step 1: Write WebSocket message types**

CREATE `/home/ilan/whatsapp-livetranslate-2.0/internal/services/transcription/websocket_types.go`:

```go
package transcription

// WebSocket request message
type WSTranscriptionRequest struct {
	URL          string `json:"url"`
	Model        string `json:"model"`
	Language     string `json:"language,omitempty"`
	CaptureMode  string `json:"captureMode,omitempty"`  // "full" or "first60"
	Diarization  bool   `json:"diarization,omitempty"`
}

// WebSocket response message (union type)
type WSTranscriptionMessage struct {
	// Message type
	Type    string `json:"type,omitempty"`    // "status", "download_progress", "transcription_chunk", "complete"
	Message string `json:"message,omitempty"` // Status message
	Error   string `json:"error,omitempty"`   // Error message

	// Download progress fields
	Percent        float64 `json:"percent,omitempty"`
	DownloadedMB   float64 `json:"downloaded_mb,omitempty"`
	SpeedMBPS      float64 `json:"speed_mbps,omitempty"`
	ETASeconds     int     `json:"eta_seconds,omitempty"`
	CurrentTime    float64 `json:"current_time,omitempty"`
	TargetDuration int     `json:"target_duration,omitempty"`

	// Transcription chunk fields
	Text       string `json:"text,omitempty"`
	ChunkIndex int    `json:"chunk_index,omitempty"`
	IsFinal    bool   `json:"is_final,omitempty"`

	// Completion fields
	DetectedLanguage string `json:"detected_language,omitempty"`
}

// Progress callback function type
type ProgressCallback func(msg WSTranscriptionMessage)

// VideoMetadata holds video information from /api/video-info
type VideoMetadata struct {
	Title              string `json:"title"`
	Channel            string `json:"channel"`
	DurationSeconds    int    `json:"duration_seconds"`
	DurationFormatted  string `json:"duration_formatted"`
	ViewCount          int    `json:"view_count"`
	ViewCountFormatted string `json:"view_count_formatted"`
	Thumbnail          string `json:"thumbnail"`
	IsYouTube          bool   `json:"is_youtube"`
}

// VideoMetadataResponse from /api/video-info
type VideoMetadataResponse struct {
	Success bool           `json:"success"`
	Data    *VideoMetadata `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}
```

**Step 2: Build to verify compilation**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/services/transcription/websocket_types.go
git commit -m "feat(transcription): add WebSocket and video metadata types

- Add WSTranscriptionRequest for WebSocket requests
- Add WSTranscriptionMessage for WebSocket responses
- Add VideoMetadata types for /api/video-info endpoint
- Support status, progress, chunks, and completion messages
- Add ProgressCallback type for real-time updates"
```

---

### Task 2: Implement WebSocket Transcription Client with Metadata

**Files:**
- Create: `internal/services/transcription/websocket_client.go`

**Step 1: Write WebSocket client implementation**

CREATE `/home/ilan/whatsapp-livetranslate-2.0/internal/services/transcription/websocket_client.go`:

```go
package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// GetVideoMetadata fetches video metadata from Deepgram /api/video-info endpoint
func (s *Service) GetVideoMetadata(ctx context.Context, videoURL string) (*VideoMetadata, error) {
	// Create request body
	requestBody := map[string]interface{}{
		"url": videoURL,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := s.baseURL + "/api/video-info"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var metadataResp VideoMetadataResponse
	if err := json.Unmarshal(respBody, &metadataResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !metadataResp.Success {
		return nil, fmt.Errorf("failed to get video metadata: %s", metadataResp.Error)
	}

	if metadataResp.Data == nil {
		return nil, fmt.Errorf("no metadata returned")
	}

	return metadataResp.Data, nil
}

// FormatDuration converts duration_seconds to human-readable format
// Example: 733 seconds → "12min and 13 seconds"
func FormatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}

	minutes := seconds / 60
	remainingSeconds := seconds % 60

	if minutes < 60 {
		if remainingSeconds == 0 {
			return fmt.Sprintf("%dmin", minutes)
		}
		return fmt.Sprintf("%dmin and %d seconds", minutes, remainingSeconds)
	}

	hours := minutes / 60
	remainingMinutes := minutes % 60

	if remainingMinutes == 0 && remainingSeconds == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	if remainingSeconds == 0 {
		return fmt.Sprintf("%dh and %dmin", hours, remainingMinutes)
	}
	return fmt.Sprintf("%dh, %dmin and %d seconds", hours, remainingMinutes, remainingSeconds)
}

// TranscribeViaWebSocket transcribes audio/video via WebSocket connection
// Provides real-time progress updates via callback
func (s *Service) TranscribeViaWebSocket(ctx context.Context, request WSTranscriptionRequest, progressCallback ProgressCallback) (*TranscriptionResult, error) {
	// Parse base URL and convert to WebSocket URL
	baseURL, err := url.Parse(s.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Convert http:// to ws://, https:// to wss://
	wsScheme := "ws"
	if baseURL.Scheme == "https" {
		wsScheme = "wss"
	}

	wsURL := fmt.Sprintf("%s://%s/ws/transcribe", wsScheme, baseURL.Host)

	// Connect to WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()

	// Send transcription request
	if err := conn.WriteJSON(request); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Collect transcription chunks (silently, don't show to user yet)
	var transcriptionChunks []string
	var detectedLanguage string

	// Read messages until completion or error
	for {
		var msg WSTranscriptionMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return nil, fmt.Errorf("failed to read message: %w", err)
		}

		// Handle error messages
		if msg.Error != "" {
			return nil, fmt.Errorf("transcription error: %s", msg.Error)
		}

		// Call progress callback if provided (for download progress and status)
		if progressCallback != nil {
			progressCallback(msg)
		}

		// Handle different message types
		switch msg.Type {
		case "status":
			// Status update - handled by progress callback
			continue

		case "download_progress":
			// Download progress - handled by progress callback
			continue

		case "transcription_chunk":
			// Collect transcription chunk SILENTLY (don't show partial text)
			if msg.Text != "" {
				transcriptionChunks = append(transcriptionChunks, msg.Text)
			}

		case "complete":
			// Transcription completed
			detectedLanguage = msg.DetectedLanguage

			// Combine all chunks into final text
			finalText := strings.Join(transcriptionChunks, " ")

			return &TranscriptionResult{
				Text:             finalText,
				DetectedLanguage: detectedLanguage,
			}, nil

		default:
			// Unknown message type - ignore
			continue
		}
	}
}

// TranscriptionResult holds the final transcription result
type TranscriptionResult struct {
	Text             string
	DetectedLanguage string
}

// QuickLanguageDetection performs quick language detection using first 60 seconds
// Returns detected language code (e.g., "he", "en") or error
func (s *Service) QuickLanguageDetection(ctx context.Context, videoURL string) (string, error) {
	// Request first 60 seconds
	request := WSTranscriptionRequest{
		URL:         videoURL,
		Model:       "whisper-v3-turbo", // Fast multilingual model
		CaptureMode: "first60",
	}

	// Transcribe first 60 seconds to get language
	result, err := s.TranscribeViaWebSocket(ctx, request, nil)
	if err != nil {
		return "", fmt.Errorf("failed to transcribe for language detection: %w", err)
	}

	// Use detected language from Deepgram
	if result.DetectedLanguage != "" {
		return result.DetectedLanguage, nil
	}

	// Fallback: return empty if no detection
	return "", fmt.Errorf("no language detected from first 60 seconds")
}
```

**Step 2: Build to verify compilation**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/services/transcription/websocket_client.go
git commit -m "feat(transcription): implement WebSocket client with video metadata

- Add GetVideoMetadata for fetching video title and duration
- Add FormatDuration helper for human-readable duration
- Add TranscribeViaWebSocket with silent chunk collection
- Progress callback shows download% but not partial text
- Add QuickLanguageDetection for first 60 seconds
- Return detected language from Deepgram
- Handle errors and connection cleanup"
```

---

## Phase 2: Update Video Message Auto-Transcription

### Task 3: Update shouldTranscribe for Video Messages

**Files:**
- Modify: `internal/services/messagehandler/transcription.go:14-28`

**Step 1: Modify shouldTranscribe function**

SURGICAL EDIT for `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/transcription.go`:

```go
// REPLACE the existing shouldTranscribe function (lines 14-28):

func shouldTranscribe(msg *waProto.Message, chatJID types.JID, stateManager *transcription.StateManager) bool {
	// Check if it's an audio OR video message
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
```

**Step 2: Build to verify compilation**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/services/messagehandler/transcription.go
git commit -m "feat(transcription): extend shouldTranscribe to support video messages

- Check for both audio AND video messages
- Video messages in /tenable-enabled chats now trigger transcription
- Maintains existing audio transcription logic"
```

---

### Task 4: Implement Video Message Transcription Handler

**Files:**
- Modify: `internal/services/messagehandler/transcription.go` (add new function)

**Step 1: Add handleVideoTranscription function**

ADD to END of `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/transcription.go`:

```go
// ADD at the end of the file (after handleAudioTranscription):

func (h *WhatsMeowEventHandler) handleVideoTranscription(msg *waProto.Message, msgInfo types.MessageInfo) error {
	ctx := context.Background()

	// Send initial "Transcribing video..." status message
	senderJID := msgInfo.Chat
	if msgInfo.Chat.Server == "g.us" {
		// In groups, use appropriate participant JID
		senderJID = types.NewJID(msgInfo.Chat.User, "s.whatsapp.net")
	}

	// Create the initial status message as a reply to the video
	initialMsg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String("🎬 מתמלל סרטון..."),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:    proto.String(msgInfo.ID),
				Participant: proto.String(senderJID.String()),
			},
		},
	}

	resp, err := h.client.SendMessage(ctx, msgInfo.Chat, initialMsg)
	if err != nil {
		fmt.Printf("Failed to send 'Transcribing video...' message: %v\n", err)
		return nil // Don't fail the whole operation
	}

	fmt.Printf("Sent 'Transcribing video...' status message (ID: %s)\n", resp.ID)

	// Download video using whatsmeow's client.Download()
	videoMsg := msg.GetVideoMessage()
	videoData, err := h.client.Download(ctx, videoMsg)
	if err != nil {
		errorMsg := &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String("❌ שגיאה: לא ניתן להוריד את הסרטון"),
				ContextInfo: &waProto.ContextInfo{
					StanzaID:    proto.String(msgInfo.ID),
					Participant: proto.String(senderJID.String()),
				},
			},
		}
		editMsg := h.client.BuildEdit(msgInfo.Chat, resp.ID, errorMsg)
		h.client.SendMessage(ctx, msgInfo.Chat, editMsg)
		return fmt.Errorf("failed to download video: %w", err)
	}

	// Save video to temp file
	tmpVideoFile, err := os.CreateTemp("", "whatsapp_video_*.mp4")
	if err != nil {
		errorMsg := &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String("❌ שגיאה: לא ניתן לשמור את הסרטון"),
				ContextInfo: &waProto.ContextInfo{
					StanzaID:    proto.String(msgInfo.ID),
					Participant: proto.String(senderJID.String()),
				},
			},
		}
		editMsg := h.client.BuildEdit(msgInfo.Chat, resp.ID, errorMsg)
		h.client.SendMessage(ctx, msgInfo.Chat, editMsg)
		return fmt.Errorf("failed to create temp video file: %w", err)
	}
	defer os.Remove(tmpVideoFile.Name()) // Clean up video file
	defer tmpVideoFile.Close()

	_, err = tmpVideoFile.Write(videoData)
	if err != nil {
		return fmt.Errorf("failed to write video data: %w", err)
	}
	tmpVideoFile.Close() // Close before processing

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

	// Transcribe the video file using existing REST API service
	// (Video messages use REST API upload, URL transcription uses WebSocket)
	result, err := h.transcriptionSvc.TranscribeAudio(ctx, tmpVideoFile.Name(), language, model)
	if err != nil {
		// Silent fail - don't spam user if transcription service is down
		fmt.Printf("Video transcription failed for chat %s: %v\n", msgInfo.Chat.String(), err)
		errorMsg := &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String("❌ שגיאה: שירות התמלול לא זמין"),
				ContextInfo: &waProto.ContextInfo{
					StanzaID:    proto.String(msgInfo.ID),
					Participant: proto.String(senderJID.String()),
				},
			},
		}
		editMsg := h.client.BuildEdit(msgInfo.Chat, resp.ID, errorMsg)
		h.client.SendMessage(ctx, msgInfo.Chat, editMsg)
		return nil
	}

	// Build the final transcription message
	response := fmt.Sprintf("🎬 *תמלול סרטון:*\n\n%s", result.Text)
	if result.DetectedLanguage != "" && result.DetectedLanguage != language {
		response += fmt.Sprintf("\n\n🌐 Detected language: %s", result.DetectedLanguage)
	}

	// Edit the status message with the actual transcription
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
		fmt.Printf("Failed to edit transcription message: %v\n", err)
		return fmt.Errorf("failed to edit transcription message: %w", err)
	}

	fmt.Printf("Video transcription message edited successfully for chat %s\n", msgInfo.Chat.String())
	return nil
}
```

**Step 2: Build to verify compilation**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/services/messagehandler/transcription.go
git commit -m "feat(transcription): implement automatic video message transcription

- Download video using whatsmeow client
- Save video to temp file (mp4 format)
- Upload to Deepgram via existing REST API
- Deepgram automatically extracts audio from video
- Send status message during processing
- Edit status message with final transcription
- Clean up temp video file after processing
- Error handling with user-friendly Hebrew messages"
```

---

### Task 5: Route Video Messages to Handler

**Files:**
- Modify: `internal/services/messagehandler/event_handler.go:21-28`

**Step 1: Update handleMessage routing**

SURGICAL EDIT for `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/event_handler.go`:

```go
// FIND the shouldTranscribe check (around line 23-27) and REPLACE with:

func (h *WhatsMeowEventHandler) handleMessage(msg *waProto.Message, msgInfo types.MessageInfo) {
	// Check if this is an audio or video message that should be transcribed
	if shouldTranscribe(msg, msgInfo.Chat, h.transcriptionState) {
		// Route to appropriate handler based on message type
		if msg.GetAudioMessage() != nil {
			if err := h.handleAudioTranscription(msg, msgInfo); err != nil {
				fmt.Printf("Audio transcription error: %v\n", err)
			}
		} else if msg.GetVideoMessage() != nil {
			if err := h.handleVideoTranscription(msg, msgInfo); err != nil {
				fmt.Printf("Video transcription error: %v\n", err)
			}
		}
		return // Don't process as command
	}

	// Existing command processing logic continues...
	text := extractText(msg)
	// ... rest of existing code unchanged ...
}
```

**Step 2: Build to verify compilation**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/services/messagehandler/event_handler.go
git commit -m "feat(transcription): route video messages to transcription handler

- Check message type (audio vs video) after shouldTranscribe
- Route audio messages to handleAudioTranscription
- Route video messages to handleVideoTranscription
- Maintain existing command processing for non-media messages"
```

---

## Phase 3: Smart /transcribe Command with Video Metadata

### Task 6: Create /transcribe Command with Smart UX

**Files:**
- Create: `internal/handlers/utility/transcribe.go`

**Step 1: Write smart /transcribe command**

CREATE `/home/ilan/whatsapp-livetranslate-2.0/internal/handlers/utility/transcribe.go`:

```go
package utility

import (
	"context"
	"fmt"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
)

type TranscribeCommand struct{}

func NewTranscribeCommand() *TranscribeCommand {
	return &TranscribeCommand{}
}

func (c *TranscribeCommand) Execute(ctx *framework.Context) error {
	if len(ctx.Args) == 0 {
		return ctx.Handler.SendResponse(ctx.MessageInfo, framework.Error("Please provide a video URL to transcribe"))
	}

	url := ctx.Args[0]
	fmt.Printf("[TRANSCRIBE] Starting transcription for URL: %s\n", url)

	// Get transcription service from handler
	transcriptionSvc := ctx.Handler.(interface {
		GetTranscriptionService() *transcription.Service
	}).GetTranscriptionService()

	// Step 1: Fetch video metadata FIRST
	fmt.Printf("[TRANSCRIBE] Step 1: Fetching video metadata...\n")
	initialMsg := framework.Processing("🎬 מתחיל תמלול...\n\n🔍 מאחזר מידע על הסרטון...")
	err := ctx.Handler.SendResponse(ctx.MessageInfo, initialMsg)
	if err != nil {
		fmt.Printf("[TRANSCRIBE] Failed to send initial message: %v\n", err)
	}

	metadata, err := transcriptionSvc.GetVideoMetadata(context.Background(), url)
	var videoTitle string
	var videoDuration string
	var videoDurationSeconds int

	if err != nil {
		fmt.Printf("[TRANSCRIBE] Failed to get video metadata: %v (continuing without metadata)\n", err)
		videoTitle = "Unknown Video"
		videoDuration = "unknown length"
	} else {
		videoTitle = metadata.Title
		videoDurationSeconds = metadata.DurationSeconds
		videoDuration = transcription.FormatDuration(metadata.DurationSeconds)
		fmt.Printf("[TRANSCRIBE] Video metadata: title=%s, duration=%s (%ds)\n", videoTitle, videoDuration, videoDurationSeconds)
	}

	// Step 2: Quick language detection (first 60 seconds)
	fmt.Printf("[TRANSCRIBE] Step 2: Detecting language from first 60 seconds...\n")
	statusMsg := fmt.Sprintf("🎬 מתחיל תמלול...\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s\n\n🔍 מזהה שפת הסרטון...", videoTitle, videoDuration)
	ctx.Handler.EditMessage(ctx.MessageInfo, statusMsg)

	detectedLangCode, err := transcriptionSvc.QuickLanguageDetection(context.Background(), url)
	if err != nil {
		fmt.Printf("[TRANSCRIBE] Language detection failed, defaulting to Hebrew: %v\n", err)
		detectedLangCode = "he" // Default to Hebrew if detection fails
	}

	fmt.Printf("[TRANSCRIBE] Detected language code from Deepgram: %s\n", detectedLangCode)

	// Step 3: Determine optimal model based on detected language
	var model string
	var languageName string

	if detectedLangCode == "he" || detectedLangCode == "iw" {
		// Hebrew detected - use ivrit-ct2 for best quality
		model = "ivrit-ct2"
		languageName = "Hebrew"
		fmt.Printf("[TRANSCRIBE] Hebrew detected, using ivrit-ct2 model\n")
	} else {
		// Non-Hebrew detected - use Deepgram for best quality
		model = "deepgram"
		languageName = detectedLangCode
		fmt.Printf("[TRANSCRIBE] Non-Hebrew detected (%s), using Deepgram model\n", detectedLangCode)
	}

	// Step 4: Show verbose progress message to user
	verboseMsg := fmt.Sprintf("🎬 Transcribing now \"%s\" length: %s and Video Language detected: %s",
		videoTitle,
		videoDuration,
		languageName)
	ctx.Handler.EditMessage(ctx.MessageInfo, verboseMsg)
	fmt.Printf("[TRANSCRIBE] Verbose message shown to user\n")

	// Step 5: Full transcription with optimal model using WebSocket
	fmt.Printf("[TRANSCRIBE] Step 3: Starting full transcription with model: %s\n", model)

	request := transcription.WSTranscriptionRequest{
		URL:         url,
		Model:       model,
		CaptureMode: "full",
	}

	// Progress callback - ONLY show download progress, NOT partial transcription
	progressCallback := func(msg transcription.WSTranscriptionMessage) {
		switch msg.Type {
		case "status":
			fmt.Printf("[TRANSCRIBE] Status: %s\n", msg.Message)
			// Update user with major status changes
			if msg.Message == "Starting transcription..." {
				statusUpdate := fmt.Sprintf("🎬 Transcribing now \"%s\"...\n\n🎙️ מתמלל...", videoTitle)
				ctx.Handler.EditMessage(ctx.MessageInfo, statusUpdate)
			}

		case "download_progress":
			// Update with download progress (every 25%)
			if msg.Percent > 0 && int(msg.Percent)%25 == 0 {
				statusUpdate := fmt.Sprintf("🎬 Transcribing now \"%s\"...\n\n📥 Downloading: %.0f%%", videoTitle, msg.Percent)
				ctx.Handler.EditMessage(ctx.MessageInfo, statusUpdate)
				fmt.Printf("[TRANSCRIBE] Download progress: %.0f%%\n", msg.Percent)
			}

		case "transcription_chunk":
			// DO NOT SHOW PARTIAL TEXT - collect silently
			fmt.Printf("[TRANSCRIBE] Received chunk %d (collecting silently)\n", msg.ChunkIndex)
		}
	}

	// Transcribe via WebSocket with progress updates
	result, err := transcriptionSvc.TranscribeViaWebSocket(context.Background(), request, progressCallback)
	if err != nil {
		fmt.Printf("[TRANSCRIBE] Transcription failed: %v\n", err)
		errorMsg := framework.Error(fmt.Sprintf("❌ שגיאה בתמלול: %v", err))
		ctx.Handler.EditMessage(ctx.MessageInfo, errorMsg)
		return nil
	}

	// Step 6: Build final response with complete transcription and metadata
	finalResponse := fmt.Sprintf("🎬 *תמלול הושלם*\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s\n🔤 שפה: %s\n\n📝 *תמלול:*\n\n%s",
		videoTitle,
		videoDuration,
		languageName,
		result.Text)

	if result.DetectedLanguage != "" {
		finalResponse += fmt.Sprintf("\n\n🌐 Confirmed language: %s", result.DetectedLanguage)
	}

	// Show final complete transcription
	ctx.Handler.EditMessage(ctx.MessageInfo, finalResponse)
	fmt.Printf("[TRANSCRIBE] Transcription complete for URL: %s (model: %s, language: %s, duration: %ds)\n",
		url, model, detectedLangCode, videoDurationSeconds)

	return nil
}

func (c *TranscribeCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "transcribe",
		Aliases:     []string{"tc"},
		Description: "Transcribe video from URL with automatic language detection",
		Category:    "Utility",
		Usage:       "/transcribe <url>",
		Examples: []string{
			"/transcribe https://www.youtube.com/shorts/Ibu3c4hkNoo?feature=share",
			"/tc https://www.youtube.com/watch?v=...",
			"/transcribe https://www.instagram.com/reel/...",
		},
		Parameters: []framework.Parameter{
			{
				Name:        "url",
				Type:        framework.StringParam,
				Description: "Video URL to transcribe (YouTube, Instagram, Twitter, etc.)",
				Required:    true,
			},
		},
		RequireOwner: false, // Available to all users
	}
}
```

**Step 2: Build to verify compilation**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/handlers/utility/transcribe.go
git commit -m "feat(transcription): add smart /transcribe command with video metadata

- Fetch video metadata FIRST (title, duration)
- Show verbose message: Transcribing now \"[Title]\" length: [duration] and Video Language detected: [language]
- Automatic language detection from first 60 seconds
- Hebrew → ivrit-ct2 model for optimal Hebrew quality
- Non-Hebrew → Deepgram model for optimal quality
- Show download progress (%) but NOT partial transcription text
- Present final complete transcription with metadata
- Available to ALL users (not just owner)
- Supports YouTube, Instagram, Twitter, TikTok, and more"
```

---

### Task 7: Register /transcribe Command

**Files:**
- Modify: `internal/services/messagehandler/event_handler.go` (InitializeCommands function)

**Step 1: Add import (if not already present)**

ADD to imports in `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/event_handler.go`:

```go
// If not already imported:
import (
	// ... existing imports ...
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/handlers/utility"
)
```

**Step 2: Register command**

ADD to InitializeCommands() function (around line 110, after existing utility commands):

```go
	// Register transcribe command
	if err := registry.Register(utility.NewTranscribeCommand()); err != nil {
		return fmt.Errorf("failed to register transcribe command: %w", err)
	}

// NOTE: Do NOT add "transcribe" to ownerCommands slice - it should be available to all users
```

**Step 3: Build to verify compilation**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/services/messagehandler/event_handler.go
git commit -m "feat(transcription): register /transcribe command

- Add command registration in InitializeCommands
- Available to all users (not owner-only)
- Command accessible via /transcribe or /tc alias"
```

---

## Phase 4: Documentation Updates

### Task 8: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Update documentation**

SURGICAL EDIT for `/home/ilan/whatsapp-livetranslate-2.0/CLAUDE.md`:

```markdown
// FIND the transcription section and REPLACE with:

## Audio & Video Transcription Architecture

The bot supports automatic audio and video message transcription with per-chat enable/disable controls, plus intelligent on-demand URL-based transcription with video metadata display:

**Transcription Flow (Auto Mode - Video Messages):**
1. Audio/video message received
2. Check if transcription enabled for chat (SQLite lookup in `transcription_settings` table)
3. If enabled:
   - Download media using `whatsmeow.Client.Download()`
   - Save to temporary file (.ogg for audio, .mp4 for video)
   - Upload to Deepgram via `POST /api/upload` → get `file_id`
   - POST to Deepgram REST endpoint: `{url: "/app/uploads/file_id", language: "he"}`
   - Deepgram automatically extracts audio from video files using ffmpeg
   - Parse response and send transcription as reply
   - Clean up temp files

**Transcription Flow (On-Demand via /transcribe - Smart UX):**
1. User sends `/transcribe [url]`
2. **Step 1: Fetch video metadata**
   - POST to `/api/video-info` to get title, duration, channel
   - Show user: "Found video: "[title]" (duration)"
3. **Step 2: Quick language detection (60 seconds)**
   - WebSocket to Deepgram: `ws://localhost:8009/ws/transcribe`
   - Request: `{url: "...", model: "whisper-v3-turbo", captureMode: "first60"}`
   - Wait for completion with detected language
   - Deepgram returns detected language in completion message
4. **Step 3: Verbose progress message**
   - Format duration: "12min and 13 seconds"
   - Show: "Transcribing now "[Title]" length: [duration] and Video Language detected: [language]"
5. **Step 4: Full transcription with optimal model**
   - If Hebrew detected → use `ivrit-ct2` model (best for Hebrew)
   - If non-Hebrew detected → use `deepgram` model (cloud quality)
   - WebSocket request: `{url: "...", model: "...", captureMode: "full"}`
   - Show download progress (%) but NOT partial transcription text
   - Collect transcription chunks silently
6. **Step 5: Present final complete transcription**
   - Show: "תמלול הושלם" with full metadata (title, duration, language)
   - Display complete transcription text
   - User gets clean, complete result
7. Deepgram handles entire workflow:
   - Download video from URL using yt-dlp
   - Extract audio using ffmpeg
   - Transcribe audio
   - Return transcription text
8. Available in ANY chat (no /tenable required)

**Commands:**
- `/tenable [language]` - Enable auto-transcription for audio AND video messages (default language: Hebrew, owner-only)
- `/tdisable` - Disable auto-transcription (owner-only)
- `/transcribe [url]` - Transcribe video from URL with smart UX (available to all users)
  - Shows video title and duration before transcribing
  - Shows detected language and model selection
  - Displays download progress (%) but not partial text
  - Presents final complete transcription with metadata

**Smart UX Features:**
- **Video metadata display** - Title, duration shown immediately
- **Verbose progress** - User knows what to expect (video length)
- **Language detection** - Automatic, shows detected language
- **Optimal model selection** - ivrit-ct2 (Hebrew) vs Deepgram (non-Hebrew)
- **Clean final result** - No confusing partial text updates
- **Progress transparency** - Download % shown, transcription collected silently

**Supported Video Sources (via Deepgram's yt-dlp):**
- YouTube (full videos, shorts, live streams)
- Instagram (reels, videos)
- Twitter/X (videos)
- TikTok (videos)
- Facebook (videos)
- And 100+ other sites supported by yt-dlp

**Deepgram Container Integration:**
- Bot uses REST API at `localhost:8009` (for video messages and metadata)
- Bot uses WebSocket at `ws://localhost:8009/ws/transcribe` (for URL transcription)
- Endpoints:
  - `POST /api/video-info` - Get video metadata (title, duration, channel)
  - `POST /api/upload` - Upload file, returns file_id
  - `POST /api/transcribe/whisper-ivrit` - Hebrew transcription (ivrit-ct2 model)
  - `POST /api/transcribe/whisper` - Standard Whisper (whisper-v3-turbo model)
  - `POST /api/transcribe/deepgram` - Deepgram cloud transcription
  - `WS /ws/transcribe` - Real-time WebSocket transcription with progress updates
- WebSocket provides real-time status, download progress, and streaming chunks
- Deepgram handles video-to-audio extraction automatically
```

**Step 2: Build to verify no compilation issues**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs(transcription): update CLAUDE.md with smart UX features

- Document video metadata fetching and display
- Explain verbose progress message format
- Document no partial text streaming (clean UX)
- List smart UX features (metadata, progress, final result)
- Add /api/video-info endpoint documentation
- Clarify 5-step transcription flow with metadata"
```

---

### Task 9: Update README.md

**Files:**
- Modify: `README.md`

**Step 1: Update README**

SURGICAL EDIT for `/home/ilan/whatsapp-livetranslate-2.0/README.md`:

```markdown
// FIND the "Transcription Commands" section and REPLACE with:

### Transcription Commands
- `/tenable [language]` - Enable auto-transcription for audio AND video messages (owner only)
- `/tdisable` - Disable auto-transcription (owner only)
- `/transcribe [url]` - **Smart video transcription** from URL with automatic language detection (available to all users)
  - **📹 Shows video title and duration** - Fetches metadata before transcribing
  - **🔍 Automatic language detection** - Analyzes first 60 seconds to detect language
  - **💬 Verbose progress** - "Transcribing now "[Title]" length: [duration] and Video Language detected: [language]"
  - **🇮🇱 Hebrew videos** → Uses ivrit-ct2 model (Hebrew-optimized, highest quality)
  - **🌍 Non-Hebrew videos** → Uses Deepgram Nova-3 model (cloud quality)
  - **📊 Smart progress updates** → Shows download % but NOT confusing partial text
  - **✅ Clean final result** → Presents complete transcription with metadata when done
  - **🎯 Smart model selection** - Automatically picks the best model for detected language
  - Example: `/transcribe https://www.youtube.com/shorts/Ibu3c4hkNoo?feature=share`
  - Example: `/transcribe https://www.instagram.com/reel/...`
  - Supports YouTube, Instagram, Twitter, TikTok, Facebook, and 100+ other platforms

// FIND the "Core Capabilities" section and UPDATE:

### Core Capabilities
- **🌐 Real-time Translation**: Translate messages between 20+ languages using Google's Gemini AI
- **🎤 Audio & Video Transcription**:
  - **📹 Automatic transcription** of voice and video messages with per-chat controls
  - **🔗 Smart URL transcription** with video metadata display (title, duration)
  - **🧠 Intelligent UX**: Shows what to expect before transcribing
  - **💬 Verbose progress**: "Transcribing now "[Title]" length: [duration] and Video Language detected: [language]"
  - **📊 Clean progress updates**: Download % shown, NOT confusing partial text
  - **✅ Complete results**: Final transcription presented with full metadata
  - **🎯 Automatic model selection**: Hebrew → ivrit-ct2, Non-Hebrew → Deepgram Nova-3
  - **⚡ Real-time detection**: First 60 seconds analyzed to detect language automatically
  - **📡 Powered by Deepgram**: Container with yt-dlp and ffmpeg integration
  - **🌍 Universal support**: YouTube, Instagram, Twitter, TikTok, and 100+ platforms
- **📥 Media Downloader**: Download videos and images from YouTube, Instagram, Twitter, and more
```

**Step 2: Build to verify no compilation issues**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs(transcription): update README.md with smart UX features

- Document video metadata display (title, duration)
- Highlight verbose progress message
- Emphasize no partial text streaming (clean UX)
- Add smart progress updates explanation
- Document complete final result presentation
- Add comprehensive UX feature bullets
- Include example message format"
```

---

## Phase 5: Final Verification and Git Push

### Task 10: Final Build Verification

**Step 1: Clean build**

```bash
cd /home/ilan/whatsapp-livetranslate-2.0
go mod tidy
go build -o whatsapp-livetranslate .
```

Expected: SUCCESS with no errors

**Step 2: Verify all files created/modified**

Run:
```bash
git status
```

Expected to show:
- Created: `internal/services/transcription/websocket_types.go`
- Created: `internal/services/transcription/websocket_client.go`
- Created: `internal/handlers/utility/transcribe.go`
- Modified: `internal/services/messagehandler/transcription.go`
- Modified: `internal/services/messagehandler/event_handler.go`
- Modified: `CLAUDE.md`
- Modified: `README.md`

**Step 3: Verify go.mod dependencies**

Check that gorilla/websocket is present:
```bash
grep gorilla/websocket go.mod
```

Expected: `github.com/gorilla/websocket v1.5.3`

**Step 4: Review all changes**

```bash
git diff --stat
```

Review all modified files to ensure changes are correct.

**Step 5: Commit final verification**

```bash
git add .
git commit -m "build(transcription): complete smart video transcription with WebSocket

✨ Features Implemented:
- WebSocket client for real-time transcription
- Video metadata fetching (title, duration) before transcription
- Verbose progress: \"Transcribing now \"[Title]\" length: [duration] and Video Language detected: [language]\"
- Automatic language detection from first 60 seconds
- Smart model selection: ivrit-ct2 (Hebrew) vs Deepgram (non-Hebrew)
- Clean UX: Download % shown, NO partial text streaming
- Final complete transcription with full metadata
- Video message auto-transcription in /tenable chats
- /transcribe command for URL-based transcription (all users)

📁 Files Added:
- internal/services/transcription/websocket_types.go (with VideoMetadata types)
- internal/services/transcription/websocket_client.go (with GetVideoMetadata)
- internal/handlers/utility/transcribe.go (smart UX implementation)

📝 Files Modified:
- internal/services/messagehandler/transcription.go (video support)
- internal/services/messagehandler/event_handler.go (routing + registration)
- CLAUDE.md (smart UX documentation)
- README.md (comprehensive feature list)

✅ Build Verification:
- go mod tidy completed
- go build successful
- All dependencies verified
- Ready for manual testing"
```

---

### Task 11: Push Changes to Multi Branch

**Step 1: Verify current branch**

```bash
git branch
```

Expected: Should show `* multi` (current branch)

If not on multi branch:
```bash
git checkout multi
```

**Step 2: Pull latest changes from remote**

```bash
git pull origin multi
```

Expected: Already up to date OR merge successful

**Step 3: Push all commits to remote**

```bash
git push origin multi
```

Expected:
```
Counting objects: X, done.
Delta compression using up to Y threads.
Compressing objects: 100% (X/X), done.
Writing objects: 100% (X/X), Z KiB | A MiB/s, done.
Total X (delta Y), reused Z (delta W)
To github.com:user/whatsapp-livetranslate-2.0.git
   abc1234..def5678  multi -> multi
```

**Step 4: Verify push was successful**

```bash
git log origin/multi -1 --oneline
```

Expected: Should show your latest commit

**Step 5: Document completion**

```bash
echo "✅ Smart video transcription feature with WebSocket complete - $(date)" >> PROGRESS.md
echo "" >> PROGRESS.md
echo "## Video Transcription Features:" >> PROGRESS.md
echo "- Implemented WebSocket client for real-time transcription" >> PROGRESS.md
echo "- Added video metadata fetching (title, duration)" >> PROGRESS.md
echo "- Verbose progress: Transcribing now \"[Title]\" length: [duration] and Video Language detected: [language]" >> PROGRESS.md
echo "- Automatic language detection (first 60 seconds)" >> PROGRESS.md
echo "- Smart model selection: ivrit-ct2 vs Deepgram" >> PROGRESS.md
echo "- Clean UX: Download progress shown, NO partial text streaming" >> PROGRESS.md
echo "- Final complete transcription with metadata" >> PROGRESS.md
echo "- Supports 100+ video platforms via yt-dlp" >> PROGRESS.md
echo "" >> PROGRESS.md

git add PROGRESS.md
git commit -m "docs: update PROGRESS.md with smart video transcription completion"
git push origin multi
```

---

## Manual Testing Guide

### Prerequisites

1. **Ensure Deepgram container is running:**
   ```bash
   docker ps | grep deepgram
   # OR check health
   curl http://localhost:8009/health
   ```
   Expected: `{"status":"healthy"...}`

2. **Test video-info endpoint:**
   ```bash
   curl -X POST http://localhost:8009/api/video-info \
     -H "Content-Type: application/json" \
     -d '{"url": "https://www.youtube.com/shorts/Ibu3c4hkNoo?feature=share"}'
   ```
   Expected: `{"success": true, "data": {"title": "...", "duration_seconds": 123, ...}}`

3. **Ensure bot is running:**
   ```bash
   cd /home/ilan/whatsapp-livetranslate-2.0
   ./whatsapp-livetranslate
   ```

### Test Scenario 1: Smart /transcribe with Video Metadata

**Test 1.1: Hebrew YouTube video**
```
User → Bot: /transcribe https://www.youtube.com/shorts/[hebrew-video-id]

Expected message sequence:
1. "🎬 מתחיל תמלול...\n\n🔍 מאחזר מידע על הסרטון..."

2. "🎬 מתחיל תמלול...\n\n📹 סרטון: \"[Video Title]\"\n⏱️ משך: 2min and 35 seconds\n\n🔍 מזהה שפת הסרטון..."

3. "🎬 Transcribing now \"[Video Title]\" length: 2min and 35 seconds and Video Language detected: Hebrew"

4. "🎬 Transcribing now \"[Video Title]\"...\n\n📥 Downloading: 25%"

5. "🎬 Transcribing now \"[Video Title]\"...\n\n📥 Downloading: 50%"

6. "🎬 Transcribing now \"[Video Title]\"...\n\n🎙️ מתמלל..."

7. "🎬 *תמלול הושלם*\n\n📹 סרטון: \"[Video Title]\"\n⏱️ משך: 2min and 35 seconds\n🔤 שפה: Hebrew\n\n📝 *תמלול:*\n\n[Full Hebrew transcription here]"
```

**Test 1.2: English YouTube video**
```
User → Bot: /transcribe https://www.youtube.com/watch?v=[english-video-id]

Expected:
- Video title and duration shown
- "Video Language detected: en"
- "Using Deepgram for optimal quality"
- Final transcription in English with Deepgram model
```

**Test 1.3: Verify NO partial text streaming**
```
User → Bot: /transcribe [long video URL]

Verify that:
- Download progress is shown (25%, 50%, 75%, 100%)
- NO partial transcription text is shown during transcription
- Only final complete transcription is shown at the end
```

**Test 1.4: Verify logs show metadata**
```bash
tail -f whatsapp-bot.log | grep TRANSCRIBE

Expected:
- "[TRANSCRIBE] Step 1: Fetching video metadata..."
- "[TRANSCRIBE] Video metadata: title=..., duration=..., ...s"
- "[TRANSCRIBE] Step 2: Detecting language..."
- "[TRANSCRIBE] Verbose message shown to user"
- "[TRANSCRIBE] Received chunk X (collecting silently)"
- "[TRANSCRIBE] Transcription complete"
```

### Test Scenario 2: Duration Formatting

**Test 2.1: Short video (< 60 seconds)**
```
User → Bot: /transcribe [30-second video]
Expected: "length: 30 seconds"
```

**Test 2.2: Medium video (1-60 minutes)**
```
User → Bot: /transcribe [video ~12min 13sec]
Expected: "length: 12min and 13 seconds"
```

**Test 2.3: Long video (> 60 minutes)**
```
User → Bot: /transcribe [video ~1h 25min 30sec]
Expected: "length: 1h, 25min and 30 seconds"
```

### Test Scenario 3: Video Message Auto-Transcription

**Test 3.1: Enable transcription**
```
User → Bot: /tenable
Expected: "✅ Transcription enabled for this chat (language: Hebrew)"
```

**Test 3.2: Send video message**
```
User → Bot: [Send short video]
Expected:
1. "🎬 מתמלל סרטון..."
2. Final: "🎬 *תמלול סרטון:*\n\n[transcription]"
```

### Test Scenario 4: Error Handling

**Test 4.1: Invalid YouTube URL**
```
User → Bot: /transcribe https://www.youtube.com/watch?v=INVALIDID
Expected: Error from video-info or WebSocket
```

**Test 4.2: Non-YouTube URL (Instagram)**
```
User → Bot: /transcribe https://www.instagram.com/reel/...
Expected:
- Metadata fetch may fail (gracefully handles with "Unknown Video")
- Transcription proceeds normally
- Language detection works
```

### Completion Checklist

Before marking complete, verify:

- [ ] `/tenable` enables transcription for audio AND video messages
- [ ] Video messages auto-transcribe via REST API
- [ ] `/transcribe [url]` fetches video metadata (title, duration)
- [ ] Verbose message shows: "Transcribing now \"[Title]\" length: [duration] and Video Language detected: [language]"
- [ ] Hebrew videos use ivrit-ct2 model
- [ ] Non-Hebrew videos use Deepgram model
- [ ] Download progress (%) is shown
- [ ] Partial transcription text is NOT shown (clean UX)
- [ ] Final complete transcription presented with metadata
- [ ] Duration formatted nicely (e.g., "12min and 13 seconds")
- [ ] WebSocket connection works (check logs)
- [ ] Language detection from first 60 seconds works
- [ ] Bot handles metadata fetch failures gracefully
- [ ] Temp files are cleaned up
- [ ] Audio transcription still works (no regression)
- [ ] Other commands still work (no regression)
- [ ] Changes pushed to multi branch successfully

---

## Success Criteria

✅ **Smart UX achieved:**
- Video title and duration shown before transcribing
- User knows what to expect (video length)
- Verbose progress message with all details
- Download progress visible (%)
- NO confusing partial text streaming
- Clean final result with complete transcription

✅ **All requirements met:**
- Video metadata fetching works
- Language detection automatic
- Model selection optimal
- Progress updates smart
- Final result complete with metadata

🎉 **Implementation complete and pushed to multi branch!**
