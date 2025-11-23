package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Service struct {
	baseURL    string
	httpClient *http.Client
}

func NewService(baseURL string) *Service {
	return &Service{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Transcription can take time
		},
	}
}

// UploadAudio uploads audio file to the service and returns the file ID
func (s *Service) UploadAudio(ctx context.Context, audioPath string) (*UploadResponse, error) {
	// Open audio file
	file, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}

	// Close multipart writer
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create upload request
	url := s.baseURL + "/api/upload"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("upload failed (no error message)")
	}

	return &result, nil
}

func (s *Service) TranscribeAudio(ctx context.Context, audioPath, language, model string) (*TranscriptionResponse, error) {
	// Step 1: Upload the audio file to get a file_id
	uploadResp, err := s.UploadAudio(ctx, audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to upload audio: %w", err)
	}

	// Step 2: Construct the file path on the transcription service's filesystem
	// The uploaded file is stored at /app/uploads/ in the service container
	// Use the file_id directly as the path since it already includes the extension
	filePath := "/app/uploads/" + uploadResp.FileID

	// Determine endpoint based on model
	endpoint := "/api/transcribe/whisper-ivrit"
	if model == "deepgram" {
		endpoint = "/api/transcribe/deepgram"
	} else if model == "whisper-v3-turbo" {
		endpoint = "/api/transcribe/whisper"
	}

	// Step 3: Create JSON request body with the file path
	requestBody := map[string]interface{}{
		"url":      filePath,
		"language": language,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create transcription request
	url := s.baseURL + endpoint
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

	// Check status code
	if resp.StatusCode != http.StatusOK {
		// Read the full response body for better error reporting
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)

		// Try to parse as structured error
		var errResp ErrorResponse
		if err := json.Unmarshal(bodyBytes, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("transcription failed: %s", errResp.Error)
		}

		// Return with full body if parsing failed
		return nil, fmt.Errorf("transcription failed with status %d: %s", resp.StatusCode, bodyString)
	}

	// Parse response
	var result TranscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("transcription failed (no error message)")
	}

	return &result, nil
}

// GetVideoMetadata fetches video metadata from the transcription service
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

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("video-info failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result VideoInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		if result.Error != "" {
			return nil, fmt.Errorf("failed to get video metadata: %s", result.Error)
		}
		return nil, fmt.Errorf("failed to get video metadata (no error message)")
	}

	return &result.Data, nil
}
