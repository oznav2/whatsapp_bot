package transcription

import "fmt"

// WSTranscriptionRequest represents the WebSocket request payload
type WSTranscriptionRequest struct {
	URL          string `json:"url,omitempty"`
	UploadFileID string `json:"upload_file_id,omitempty"`
	Language     string `json:"language,omitempty"`
	Model        string `json:"model,omitempty"`
	CaptureMode  string `json:"captureMode,omitempty"` // "first60" or "full"
}

// WSTranscriptionMessage represents messages received from the WebSocket
type WSTranscriptionMessage struct {
	Type    string `json:"type,omitempty"`    // "status", "download_progress", "transcription_chunk", "complete", "error"
	Message string `json:"message,omitempty"` // Human-readable status message

	// Download progress fields
	Percent      float64 `json:"percent,omitempty"`
	DownloadedMB float64 `json:"downloaded_mb,omitempty"`
	TotalMB      float64 `json:"total_mb,omitempty"`

	// Transcription chunk fields
	Text       string `json:"text,omitempty"`
	ChunkIndex int    `json:"chunk_index,omitempty"`

	// Completion fields
	DetectedLanguage string `json:"detected_language,omitempty"`
	FinalText        string `json:"final_text,omitempty"`

	// Error fields
	Error string `json:"error,omitempty"`
}

// VideoMetadata represents metadata about a video
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

// VideoInfoResponse represents the response from /api/video-info
type VideoInfoResponse struct {
	Success bool          `json:"success"`
	Data    VideoMetadata `json:"data"` // VibeGram returns "data", not "metadata"
	Error   string        `json:"error,omitempty"`
}

// FormatDuration converts seconds to human-readable format
// Examples: "45 seconds", "12min and 13 seconds", "1h, 23min and 45 seconds"
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
