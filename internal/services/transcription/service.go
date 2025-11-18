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

func (s *Service) TranscribeAudio(ctx context.Context, audioPath, language, model string) (*TranscriptionResponse, error) {
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

	// Add language field
	if language != "" {
		writer.WriteField("language", language)
	}

	// Close multipart writer
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Determine endpoint based on model
	endpoint := "/api/transcribe/whisper-ivrit"
	if model == "deepgram" {
		endpoint = "/api/transcribe/deepgram"
	} else if model == "whisper-v3-turbo" {
		endpoint = "/api/transcribe/whisper"
	}

	// Create request
	url := s.baseURL + endpoint
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
