package transcription

type TranscriptionResponse struct {
	Success          bool     `json:"success"`
	Status           string   `json:"status"`
	Model            string   `json:"model"`
	Language         string   `json:"language"`
	Text             string   `json:"text"`
	Confidence       *float64 `json:"confidence"`
	DetectedLanguage string   `json:"detected_language"`
	Timings          Timings  `json:"timings"`
}

type Timings struct {
	DownloadMs   int `json:"download_ms"`
	TranscribeMs int `json:"transcribe_ms"`
	TotalMs      int `json:"total_ms"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type UploadResponse struct {
	Success  bool    `json:"success"`
	FileID   string  `json:"file_id"`
	Filename string  `json:"filename"`
	SizeMB   float64 `json:"size_mb"`
}
