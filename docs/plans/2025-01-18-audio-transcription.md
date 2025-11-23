# Audio Transcription Feature Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add automatic audio message transcription with per-chat enable/disable controls, using the existing live transcription container at localhost:8009.

---

## 🚨 CRITICAL IMPLEMENTATION GUIDELINES

### ❌ ABSOLUTE PROHIBITIONS - Docker Builds

**Throughout the ENTIRE implementation process (Phases 1-8), you are EXPLICITLY FORBIDDEN from:**

- ❌ Running `docker build`
- ❌ Running `docker-compose build`
- ❌ Running `docker run`
- ❌ Testing in Docker containers
- ❌ Suggesting Docker builds for verification
- ❌ Checking if Dockerfile changes work

**WHY:**
- Application will be non-functional until ALL phases complete
- Docker builds are time-consuming and unnecessary during development
- Module imports will fail until full structure exists
- Container builds consume tokens without value
- **User will handle Docker build ONCE after ALL implementation phases are complete**

### 💾 Token-Saving Strategies (MANDATORY)

#### 1. Use Diff-Style Instructions for Go File Modifications

**❌ NEVER DO THIS:**
```go
// Regenerating entire main.go (3,618 lines)
package main

import (
    // ... 50 imports ...
)

func main() {
    // ... 3,500 lines of code ...
}
```

**✅ ALWAYS DO THIS:**
```go
// SURGICAL EDIT for main.go:22-47

// ADD after line 35:
import (
    "github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
)

// REPLACE lines 45-47:
- evtHandler, err := messagehandler.NewWhatsMeowEventHandler(client, detector, translator, imageGenerator)
+ transcriptionState := transcription.NewStateManager(container.DB)
+ transcriptionSvc := transcription.NewService(config.AppConfig.TranscribeServiceURL)
+ evtHandler, err := messagehandler.NewWhatsMeowEventHandler(client, detector, translator, imageGenerator, transcriptionState, transcriptionSvc)
```

**Token Savings:** Don't regenerate 3,618 lines of code. Only show the 10-20 lines that change.

#### 2. Pre-Built Import Statements (Don't Make LLM Figure Out)

**Instead of:** Letting LLM determine what to import in each module.

**Do this:** Pre-calculate imports for each module in your plan:

```go
// internal/services/transcription/state_manager.go
// REQUIRED IMPORTS (copy exactly):
import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "go.mau.fi/whatsmeow/types"
)

// internal/handlers/transcription/tenable.go
// REQUIRED IMPORTS (copy exactly):
import (
    framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
    "github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
)
```

**Token Savings:** ~200 tokens per file × 15 files = 3,000 tokens saved

#### 3. Verification Strategy: Once Per Phase (Not Per Task)

**❌ WRONG APPROACH:**
- Task 1 complete → Run `go test` → Run `go build` → Run `gofmt`
- Task 2 complete → Run `go test` → Run `go build` → Run `gofmt`
- Task 3 complete → Run `go test` → Run `go build` → Run `gofmt`

**✅ CORRECT APPROACH:**
- Complete ALL tasks in Phase 1
- **THEN** run verification once: `go test ./internal/services/transcription/ -v && go build -o whatsapp-livetranslate .`

**Token Savings:** One verification at phase end vs. 5-10 verifications during phase

### 🏗️ Modular Architecture (MANDATORY)

**TARGET: Each functionality in its own file. No redundant files unless necessary.**

#### File Organization Principles

**1. One Responsibility Per File**

Each file should have a single, clear purpose:

✅ **GOOD:**
```
internal/services/transcription/
├── state_manager.go          # State management only
├── state_manager_test.go     # Tests for state manager
├── service.go                # HTTP client only
├── service_test.go           # Tests for HTTP client
├── models.go                 # Data structures only
└── errors.go                 # Error types (if needed)
```

❌ **BAD:**
```
internal/services/transcription/
├── transcription.go          # Everything mixed together
├── transcription_test.go     # All tests in one file
├── utils.go                  # Generic utilities (vague)
├── helpers.go                # More vague utilities
└── common.go                 # Even more mixed responsibilities
```

**2. Clear File Naming Conventions**

✅ **GOOD:**
- `state_manager.go` - Manages transcription state in database
- `service.go` - HTTP client for transcription API
- `models.go` - Request/response data structures
- `tenable.go` - Command to enable transcription
- `tdisable.go` - Command to disable transcription

❌ **BAD:**
- `utils.go` - Too vague, what utilities?
- `helpers.go` - Too generic, helpers for what?
- `common.go` - What's common?
- `stuff.go` - Meaningless name
- `transcription_utils_helpers_common.go` - Too long, unclear

**3. Avoid Redundant Files**

❌ **DON'T CREATE:**
- `internal/services/transcription/transcription_service.go` (redundant "transcription" in path and filename)
- `internal/handlers/transcription/transcription_handler.go` (redundant naming)
- `internal/services/transcription/transcription_models.go` (just use `models.go`)
- Empty interface files with no purpose
- Wrapper files that just call another file

✅ **DO CREATE:**
- `internal/services/transcription/service.go` (clear, no redundancy)
- `internal/handlers/transcription/tenable.go` (specific command)
- `internal/services/transcription/models.go` (clear purpose)

**4. Logical Grouping by Functionality**

```
internal/
├── services/
│   └── transcription/          # Service layer
│       ├── state_manager.go    # Database state management
│       ├── service.go          # HTTP API client
│       └── models.go           # Data structures
├── handlers/
│   └── transcription/          # Command handlers
│       ├── tenable.go          # Enable command
│       └── tdisable.go         # Disable command
└── messagehandler/
    ├── audio_utils.go          # Audio message helpers
    └── transcription.go        # Audio transcription flow
```

**5. When to Split vs. When to Combine**

**SPLIT into separate files when:**
- Two functionalities serve different purposes (state management vs HTTP client)
- File exceeds ~300-400 lines
- Different teams might work on different parts
- Testing requires different mocks/fixtures

**KEEP in same file when:**
- Functions are tightly coupled and always used together
- Total lines < 200 and single responsibility
- Splitting would require excessive imports between files
- Helper functions only used by one main function

**6. Test File Organization**

✅ **GOOD:**
- `state_manager_test.go` - Tests for `state_manager.go` only
- `service_test.go` - Tests for `service.go` only
- `tenable_test.go` - Tests for `tenable.go` only

❌ **BAD:**
- `transcription_test.go` - All tests mixed together
- `all_tests.go` - Generic test file
- Spreading tests for one file across multiple test files

### 📊 Progress Tracking (MANDATORY)

**Create `/home/ilan/whatsapp-livetranslate-2.0/PROGRESS.md` at start of implementation:**

```markdown
# Audio Transcription Feature - Implementation Progress

**Started:** [Date]
**Status:** In Progress

## Phase Completion

- [ ] Phase 1: Database Schema and State Management
- [ ] Phase 2: Transcription Service HTTP Client
- [ ] Phase 3: Audio Download Helper
- [ ] Phase 4: Transcription Commands
- [ ] Phase 5: Event Handler Integration
- [ ] Phase 6: Configuration and Documentation
- [ ] Phase 7: Testing and Verification
- [ ] Phase 8: Final Integration and Cleanup

## Phase Details

### Phase 1: Database Schema and State Management
**Status:** Not Started
**Tasks:**
- [ ] Task 1: Create Transcription Settings Schema

### Phase 2: Transcription Service HTTP Client
**Status:** Not Started
**Tasks:**
- [ ] Task 2: Create Transcription Service

... (continue for all phases)
```

**Update PROGRESS.md after EACH phase completion:**

```markdown
### Phase 1: Database Schema and State Management ✅
**Status:** COMPLETED
**Completed:** 2025-01-18 14:30
**Tasks:**
- [x] Task 1: Create Transcription Settings Schema
**Tests Passing:** ✅ 3/3
**Files Modified:** 3 files created
**Commit:** feat(transcription): add state manager with SQLite persistence (abc123)
**Notes:** All tests passing, schema working as expected
```

**Benefits:**
- User can monitor progress in real-time
- Complete phase-by-phase record
- Easy to resume if interrupted
- Documentation of what was actually completed

---

**Architecture:** Event-driven audio processing integrated into the message handler flow. Audio messages are detected (including PTT voice notes), downloaded via whatsmeow's `client.Download()`, uploaded to the transcription REST API, and responses are sent back to WhatsApp. Per-chat settings stored in existing SQLite database (auth.db). Default language is Hebrew with fallback language detection.

**Tech Stack:** Go, whatsmeow, SQLite3 (existing), HTTP REST client, VibeGram transcription service (Whisper/Ivrit/Deepgram models)

**Transcription Service Details (VibeGram v5.0):**
- **Architecture**: Modular FastAPI application with 17 specialized modules
- **Port**: 8009 (default, configurable via TRANSCRIBE_SERVICE_URL)
- **Models Available**:
  - `ivrit-ct2` - Ivrit Large V3 Turbo (Hebrew-optimized, ~10GB RAM)
  - `ivrit-v3-turbo` - Alternative Ivrit model
  - `whisper-v3-turbo` - Fast multilingual Whisper (~6GB RAM)
  - `deepgram` - Cloud-based Deepgram Nova-3 (requires API key, <100ms latency)
- **Features**: Speaker diarization, deduplication, summarization, translation, audio caching
- **Processing**: Handles audio download, FFmpeg conversion, chunking internally
- **API**: REST endpoints for synchronous/async transcription with multipart file upload

**Dependencies:** NO new dependencies required! All functionality uses existing modules:
- `whatsmeow` - already has `client.Download(audioMessage)` for audio download
- `go-sqlite3` - already available for state management
- `godotenv` - already available for configuration
- Standard library `net/http`, `os`, `io` for REST client and file handling

**Proven Patterns from Reference Repositories:**
- Audio download: `client.Download(audioMessage)` returns `[]byte` directly (whatsmeow-transcribe pattern)
- PTT detection: Check `msg.GetAudioMessage().GetPTT()` for voice notes vs audio files
- Temp files: Use `os.CreateTemp("", "whatsapp_audio_*.ogg")` pattern
- Database: Reuse existing `container.DB` from sqlstore (transcribe-wa pattern)
- State schema: Simple `chat_jid` + `enabled` boolean (all repos use this)
- Error handling: Silent fail for service unavailable, log only (whatsmeow-transcribe pattern)
- Concurrent processing: Process each message in main goroutine (current bot pattern)

**Integration with VibeGram Transcription Service:**
- **Why REST not WebSocket**: Simpler for one-off transcriptions, no need for real-time progress in WhatsApp chat
- **File Upload**: VibeGram accepts multipart/form-data with audio file
- **Service Responsibilities**: VibeGram handles all audio processing (FFmpeg, chunking, model loading)
- **Bot Responsibilities**: Download from WhatsApp → Upload to VibeGram → Send response back
- **Caching**: VibeGram has built-in SHA256-based audio caching (60% CPU reduction for repeated content)
- **Models**: Use `ivrit-ct2` for Hebrew (default), `whisper-v3-turbo` for other languages
- **Error Recovery**: If VibeGram is unavailable, bot logs error and continues (no user spam)

---

## Phase 1: Database Schema and State Management

### Task 1: Create Transcription Settings Schema

**Files:**
- Create: `internal/services/transcription/schema.sql`
- Create: `internal/services/transcription/state_manager.go`
- Create: `internal/services/transcription/state_manager_test.go`

**Step 1: Write the failing test**

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/services/transcription/state_manager_test.go`:

```go
package transcription

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/types"
)

func TestStateManager_IsEnabled(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	manager := NewStateManager(db)
	ctx := context.Background()

	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}

	// Test: new chat should default to disabled
	enabled, err := manager.IsEnabled(ctx, chatJID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("expected transcription to be disabled by default")
	}
}

func TestStateManager_Enable(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	manager := NewStateManager(db)
	ctx := context.Background()

	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}

	// Enable transcription
	err = manager.Enable(ctx, chatJID, "he")
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	// Verify it's enabled
	enabled, err := manager.IsEnabled(ctx, chatJID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if !enabled {
		t.Error("expected transcription to be enabled")
	}

	// Verify language
	lang, err := manager.GetLanguage(ctx, chatJID)
	if err != nil {
		t.Fatalf("GetLanguage failed: %v", err)
	}
	if lang != "he" {
		t.Errorf("expected language 'he', got '%s'", lang)
	}
}

func TestStateManager_Disable(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	manager := NewStateManager(db)
	ctx := context.Background()

	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}

	// Enable first
	_ = manager.Enable(ctx, chatJID, "he")

	// Disable
	err = manager.Disable(ctx, chatJID)
	if err != nil {
		t.Fatalf("Disable failed: %v", err)
	}

	// Verify it's disabled
	enabled, err := manager.IsEnabled(ctx, chatJID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("expected transcription to be disabled")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/transcription/ -v`
Expected: FAIL with "no such file or directory" or "package not found"

**Step 3: Write minimal implementation**

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/services/transcription/state_manager.go`:

```go
package transcription

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/types"
)

type StateManager struct {
	db *sql.DB
}

func NewStateManager(db *sql.DB) *StateManager {
	manager := &StateManager{db: db}
	manager.initSchema()
	return manager
}

func (sm *StateManager) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS transcription_settings (
		chat_jid TEXT PRIMARY KEY,
		enabled INTEGER NOT NULL DEFAULT 0,
		language TEXT DEFAULT 'he',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_transcription_enabled
	ON transcription_settings(enabled);
	`

	_, err := sm.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

func (sm *StateManager) IsEnabled(ctx context.Context, chatJID types.JID) (bool, error) {
	var enabled int
	query := "SELECT enabled FROM transcription_settings WHERE chat_jid = ?"
	err := sm.db.QueryRowContext(ctx, query, chatJID.String()).Scan(&enabled)

	if err == sql.ErrNoRows {
		return false, nil // Default to disabled
	}
	if err != nil {
		return false, fmt.Errorf("failed to check if enabled: %w", err)
	}

	return enabled == 1, nil
}

func (sm *StateManager) Enable(ctx context.Context, chatJID types.JID, language string) error {
	query := `
	INSERT INTO transcription_settings (chat_jid, enabled, language, updated_at)
	VALUES (?, 1, ?, ?)
	ON CONFLICT(chat_jid) DO UPDATE SET
		enabled = 1,
		language = excluded.language,
		updated_at = excluded.updated_at
	`

	_, err := sm.db.ExecContext(ctx, query, chatJID.String(), language, time.Now())
	if err != nil {
		return fmt.Errorf("failed to enable transcription: %w", err)
	}
	return nil
}

func (sm *StateManager) Disable(ctx context.Context, chatJID types.JID) error {
	query := `
	INSERT INTO transcription_settings (chat_jid, enabled, updated_at)
	VALUES (?, 0, ?)
	ON CONFLICT(chat_jid) DO UPDATE SET
		enabled = 0,
		updated_at = excluded.updated_at
	`

	_, err := sm.db.ExecContext(ctx, query, chatJID.String(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to disable transcription: %w", err)
	}
	return nil
}

func (sm *StateManager) GetLanguage(ctx context.Context, chatJID types.JID) (string, error) {
	var language string
	query := "SELECT language FROM transcription_settings WHERE chat_jid = ?"
	err := sm.db.QueryRowContext(ctx, query, chatJID.String()).Scan(&language)

	if err == sql.ErrNoRows {
		return "he", nil // Default to Hebrew
	}
	if err != nil {
		return "", fmt.Errorf("failed to get language: %w", err)
	}

	return language, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/transcription/ -v`
Expected: PASS (all 3 tests)

**Step 5: Commit**

```bash
git add internal/services/transcription/
git commit -m "feat(transcription): add state manager with SQLite persistence

- Create transcription_settings table schema
- Implement Enable/Disable/IsEnabled/GetLanguage methods
- Add comprehensive unit tests
- Default language: Hebrew (he)
- Default state: disabled"
```

---

## Phase 2: Transcription Service HTTP Client

### Task 2: Create Transcription Service

**Files:**
- Create: `internal/services/transcription/service.go`
- Create: `internal/services/transcription/service_test.go`
- Create: `internal/services/transcription/models.go`

**Step 1: Write the failing test**

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/services/transcription/service_test.go`:

```go
package transcription

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestService_TranscribeAudio(t *testing.T) {
	// Create mock transcription server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/transcribe/whisper-ivrit" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("no file in form: %v", err)
		}
		defer file.Close()

		language := r.FormValue("language")
		if language != "he" {
			t.Errorf("expected language 'he', got '%s'", language)
		}

		// Return mock response
		response := TranscriptionResponse{
			Success: true,
			Status:  "ok",
			Model:   "ivrit-ct2",
			Language: "he",
			Text:    "שלום עולם",
			DetectedLanguage: "he",
			Timings: Timings{
				DownloadMs:   100,
				TranscribeMs: 2000,
				TotalMs:      2100,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create test audio file
	tmpFile, err := os.CreateTemp("", "test_audio_*.ogg")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write some dummy data
	tmpFile.Write([]byte("dummy audio data"))

	service := NewService(server.URL)
	ctx := context.Background()

	result, err := service.TranscribeAudio(ctx, tmpFile.Name(), "he", "ivrit-ct2")
	if err != nil {
		t.Fatalf("TranscribeAudio failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success = true")
	}
	if result.Text != "שלום עולם" {
		t.Errorf("expected text 'שלום עולם', got '%s'", result.Text)
	}
	if result.Language != "he" {
		t.Errorf("expected language 'he', got '%s'", result.Language)
	}
}

func TestService_TranscribeAudio_Error(t *testing.T) {
	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Transcription service unavailable",
		})
	}))
	defer server.Close()

	tmpFile, err := os.CreateTemp("", "test_audio_*.ogg")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	tmpFile.Write([]byte("dummy audio data"))

	service := NewService(server.URL)
	ctx := context.Background()

	_, err = service.TranscribeAudio(ctx, tmpFile.Name(), "he", "ivrit-ct2")
	if err == nil {
		t.Error("expected error, got nil")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/transcription/ -v -run TestService`
Expected: FAIL with "undefined: TranscriptionResponse" or similar

**Step 3: Write minimal implementation**

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/services/transcription/models.go`:

```go
package transcription

type TranscriptionResponse struct {
	Success          bool    `json:"success"`
	Status           string  `json:"status"`
	Model            string  `json:"model"`
	Language         string  `json:"language"`
	Text             string  `json:"text"`
	Confidence       *float64 `json:"confidence"`
	DetectedLanguage string  `json:"detected_language"`
	Timings          Timings `json:"timings"`
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
```

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/services/transcription/service.go`:

```go
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
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("transcription failed: %s", errResp.Error)
		}
		return nil, fmt.Errorf("transcription failed with status %d", resp.StatusCode)
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/transcription/ -v -run TestService`
Expected: PASS (2 tests)

**Step 5: Commit**

```bash
git add internal/services/transcription/service.go internal/services/transcription/service_test.go internal/services/transcription/models.go
git commit -m "feat(transcription): add HTTP client for transcription service

- Create Service with TranscribeAudio method
- Support whisper-ivrit, whisper, and deepgram endpoints
- Multipart file upload with language parameter
- Comprehensive error handling
- Add tests with mock HTTP server
- 5-minute timeout for long transcriptions"
```

---

## Phase 3: Audio Download Helper (SIMPLIFIED)

### Task 3: Add Audio Download Helper Function

**NOTE:** After analyzing reference repositories, we found that `client.Download(audioMessage)` is already available in whatsmeow and returns `[]byte` directly. No need to add to ClientInterface - we'll use it directly in the transcription handler.

**Files:**
- Create: `internal/services/messagehandler/audio_utils.go`
- Create: `internal/services/messagehandler/audio_utils_test.go`

**Step 1: Write the failing test**

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/audio_utils_test.go`:

```go
package messagehandler

import (
	"testing"

	waProto "go.mau.fi/whatsmeow/proto/waE2E"
)

func TestIsAudioMessage(t *testing.T) {
	// Test: audio message
	audioMsg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			Url: stringPtr("https://example.com/audio.ogg"),
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
			Url: stringPtr("https://example.com/voice.ogg"),
			PTT: boolPtr(true),
		},
	}

	if !isPTTVoiceNote(pttMsg) {
		t.Error("expected isPTTVoiceNote to return true for PTT message")
	}

	// Test: regular audio file
	audioMsg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			Url: stringPtr("https://example.com/audio.mp3"),
			PTT: boolPtr(false),
		},
	}

	if isPTTVoiceNote(audioMsg) {
		t.Error("expected isPTTVoiceNote to return false for non-PTT audio")
	}

	// Test: PTT not set (defaults to false)
	audioMsgNoPTT := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			Url: stringPtr("https://example.com/audio.mp3"),
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/messagehandler/ -v -run TestIsAudioMessage`
Expected: FAIL with "undefined: isAudioMessage"

**Step 3: Write minimal implementation**

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/audio_utils.go`:

```go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/messagehandler/ -v -run TestIsAudioMessage`
Expected: PASS (both tests)

**Step 5: Commit**

```bash
git add internal/services/messagehandler/audio_utils.go internal/services/messagehandler/audio_utils_test.go
git commit -m "feat(transcription): add audio message helper functions

- Add isAudioMessage to check for audio content
- Add isPTTVoiceNote to distinguish voice notes from audio files
- Based on proven pattern from whatsmeow-transcribe
- Add comprehensive unit tests for both helpers"
```

**Implementation Note:** We'll use `h.handler.client.Download(audioMsg)` directly in Task 8 when implementing the transcription flow. This is the standard whatsmeow pattern used by all reference repositories.

---

## Phase 4: Transcription Commands

### Task 4: Implement TEnableCommand

**Files:**
- Create: `internal/handlers/transcription/tenable.go`
- Create: `internal/handlers/transcription/tenable_test.go`

**Step 1: Write the failing test**

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/handlers/transcription/tenable_test.go`:

```go
package transcription

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	"go.mau.fi/whatsmeow/types"
)

type mockHandler struct {
	lastResponse string
	stateManager *transcription.StateManager
}

func (m *mockHandler) SendResponse(msgInfo types.MessageInfo, text string) error {
	m.lastResponse = text
	return nil
}

func (m *mockHandler) GetStateManager() *transcription.StateManager {
	return m.stateManager
}

// Implement other required HandlerInterface methods as no-ops
func (m *mockHandler) SendMedia(msgInfo types.MessageInfo, mediaType framework.MediaType, data []byte, caption string) error { return nil }
func (m *mockHandler) SendImage(msgInfo types.MessageInfo, upload framework.UploadResponse, caption string) error { return nil }
func (m *mockHandler) SendVideo(msgInfo types.MessageInfo, upload framework.UploadResponse, caption string) error { return nil }
func (m *mockHandler) SendDocument(msgInfo types.MessageInfo, upload framework.UploadResponse, caption string) error { return nil }
func (m *mockHandler) EditMessage(msgInfo types.MessageInfo, newText string) error { return nil }
func (m *mockHandler) EditMessageWithOriginal(msgInfo types.MessageInfo, newText string, originalMsg *framework.Message) error { return nil }
func (m *mockHandler) GetClient() framework.ClientInterface { return nil }
func (m *mockHandler) GetTranslator() framework.TranslatorInterface { return nil }
func (m *mockHandler) GetImageGenerator() framework.ImageGeneratorInterface { return nil }
func (m *mockHandler) GetMemeGenerator() framework.MemeGeneratorInterface { return nil }
func (m *mockHandler) GetLangDetector() framework.LangDetectorInterface { return nil }

func TestTEnableCommand_Execute(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	stateManager := transcription.NewStateManager(db)

	handler := &mockHandler{
		stateManager: stateManager,
	}

	cmd := NewTEnableCommand()

	ctx := &framework.Context{
		Context: context.Background(),
		MessageInfo: types.MessageInfo{
			Chat: types.JID{User: "1234567890", Server: "s.whatsapp.net"},
		},
		Command: "tenable",
		Args:    []string{},
		RawArgs: "",
		Handler: handler,
	}

	err = cmd.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify response was sent
	if handler.lastResponse == "" {
		t.Error("expected response, got empty string")
	}

	// Verify transcription was enabled
	enabled, err := stateManager.IsEnabled(context.Background(), ctx.MessageInfo.Chat)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if !enabled {
		t.Error("expected transcription to be enabled")
	}
}

func TestTEnableCommand_ExecuteWithLanguage(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	stateManager := transcription.NewStateManager(db)

	handler := &mockHandler{
		stateManager: stateManager,
	}

	cmd := NewTEnableCommand()

	ctx := &framework.Context{
		Context: context.Background(),
		MessageInfo: types.MessageInfo{
			Chat: types.JID{User: "1234567890", Server: "s.whatsapp.net"},
		},
		Command: "tenable",
		Args:    []string{"en"},
		RawArgs: "en",
		Handler: handler,
	}

	err = cmd.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify language was set
	lang, err := stateManager.GetLanguage(context.Background(), ctx.MessageInfo.Chat)
	if err != nil {
		t.Fatalf("GetLanguage failed: %v", err)
	}
	if lang != "en" {
		t.Errorf("expected language 'en', got '%s'", lang)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/handlers/transcription/ -v`
Expected: FAIL with "no such file or directory"

**Step 3: Write minimal implementation**

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/handlers/transcription/tenable.go`:

```go
package transcription

import (
	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
)

type TEnableCommand struct{}

func NewTEnableCommand() *TEnableCommand {
	return &TEnableCommand{}
}

func (c *TEnableCommand) Execute(ctx *framework.Context) error {
	// Get state manager from handler
	stateManager := ctx.Handler.(interface {
		GetStateManager() *transcription.StateManager
	}).GetStateManager()

	// Determine language (default to Hebrew)
	language := "he"
	if len(ctx.Args) > 0 {
		language = ctx.Args[0]
	}

	// Enable transcription for this chat
	err := stateManager.Enable(ctx.Context, ctx.MessageInfo.Chat, language)
	if err != nil {
		return ctx.Handler.SendResponse(
			ctx.MessageInfo,
			framework.Error("Failed to enable transcription"),
		)
	}

	// Send success message
	message := framework.Success("✅ Transcription enabled for this chat")
	if language != "he" {
		message += "\n🌐 Language: " + language
	}

	return ctx.Handler.SendResponse(ctx.MessageInfo, message)
}

func (c *TEnableCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "tenable",
		Description: "Enable audio transcription for this chat",
		Category:    "Transcription",
		Usage:       "/tenable [language]",
		Examples: []string{
			"/tenable",
			"/tenable he",
			"/tenable en",
		},
		Parameters: []framework.Parameter{
			{
				Name:        "language",
				Type:        framework.StringParam,
				Description: "Transcription language code (default: he)",
				Required:    false,
				Default:     "he",
			},
		},
		RequireOwner: true,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/handlers/transcription/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/handlers/transcription/
git commit -m "feat(transcription): add tenable command to enable transcription

- Create TEnableCommand with Execute and Metadata
- Support optional language parameter (default: Hebrew)
- Integrate with StateManager
- Owner-only command
- Add comprehensive unit tests"
```

---

### Task 5: Implement TDisableCommand

**Files:**
- Create: `internal/handlers/transcription/tdisable.go`
- Create: `internal/handlers/transcription/tdisable_test.go`

**Step 1: Write the failing test**

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/handlers/transcription/tdisable_test.go`:

```go
package transcription

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	"go.mau.fi/whatsmeow/types"
)

func TestTDisableCommand_Execute(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	stateManager := transcription.NewStateManager(db)

	// Enable transcription first
	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}
	stateManager.Enable(context.Background(), chatJID, "he")

	handler := &mockHandler{
		stateManager: stateManager,
	}

	cmd := NewTDisableCommand()

	ctx := &framework.Context{
		Context: context.Background(),
		MessageInfo: types.MessageInfo{
			Chat: chatJID,
		},
		Command: "tdisable",
		Args:    []string{},
		RawArgs: "",
		Handler: handler,
	}

	err = cmd.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify response was sent
	if handler.lastResponse == "" {
		t.Error("expected response, got empty string")
	}

	// Verify transcription was disabled
	enabled, err := stateManager.IsEnabled(context.Background(), chatJID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("expected transcription to be disabled")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/handlers/transcription/ -v -run TestTDisableCommand`
Expected: FAIL with "undefined: NewTDisableCommand"

**Step 3: Write minimal implementation**

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/handlers/transcription/tdisable.go`:

```go
package transcription

import (
	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
)

type TDisableCommand struct{}

func NewTDisableCommand() *TDisableCommand {
	return &TDisableCommand{}
}

func (c *TDisableCommand) Execute(ctx *framework.Context) error {
	// Get state manager from handler
	stateManager := ctx.Handler.(interface {
		GetStateManager() *transcription.StateManager
	}).GetStateManager()

	// Disable transcription for this chat
	err := stateManager.Disable(ctx.Context, ctx.MessageInfo.Chat)
	if err != nil {
		return ctx.Handler.SendResponse(
			ctx.MessageInfo,
			framework.Error("Failed to disable transcription"),
		)
	}

	// Send success message
	return ctx.Handler.SendResponse(
		ctx.MessageInfo,
		framework.Success("🔇 Transcription disabled for this chat"),
	)
}

func (c *TDisableCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "tdisable",
		Description: "Disable audio transcription for this chat",
		Category:    "Transcription",
		Usage:       "/tdisable",
		Examples:    []string{"/tdisable"},
		RequireOwner: true,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/handlers/transcription/ -v`
Expected: PASS (all transcription tests)

**Step 5: Commit**

```bash
git add internal/handlers/transcription/tdisable.go internal/handlers/transcription/tdisable_test.go
git commit -m "feat(transcription): add tdisable command to disable transcription

- Create TDisableCommand with Execute and Metadata
- Integrate with StateManager
- Owner-only command
- Add unit tests"
```

---

## Phase 5: Event Handler Integration

### Task 6: Add StateManager to Event Handler (UPDATED)

**NOTE:** After analyzing the current bot's main.go, we discovered that `container.DB` from `sqlstore.New()` is available but not exposed. We need to use the existing database connection.

**Files:**
- Modify: `internal/services/messagehandler/base.go:20-30`
- Modify: `internal/services/messagehandler/handler_adapter.go:15-25`
- Modify: `main.go:22-47`

**Step 1: Write the failing test**

This is an integration change, so we'll verify compilation first.

**Step 2: Run build to verify it fails**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS (no changes yet)

**Step 3: Write minimal implementation**

Modify `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/base.go`:

```go
// Add to imports:
import (
	// ... existing imports ...
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
)

// Add field to WhatsMeowEventHandler struct (around line 20):
type WhatsMeowEventHandler struct {
	client             *whatsmeow.Client
	langDetector       services.LangDetectorInterface
	translator         services.TranslateService
	imageGenerator     services.ImageGeneratorInterface
	commandRegistry    *framework.Registry
	transcriptionState *transcription.StateManager
	transcriptionSvc   *transcription.Service
}
```

Modify `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/handler_adapter.go`:

```go
// Add method to HandlerAdapter (around line 95):
func (h *HandlerAdapter) GetStateManager() *transcription.StateManager {
	return h.handler.transcriptionState
}

func (h *HandlerAdapter) GetTranscriptionService() *transcription.Service {
	return h.handler.transcriptionSvc
}
```

Modify `/home/ilan/whatsapp-livetranslate-2.0/main.go`:

```go
// Add to imports:
import (
	// ... existing imports ...
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
)

// In main() function, IMPORTANT: use container.DB (around line 22-47):
func main() {
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:/data/auth.db?_foreign_keys=on", nil)
	if err != nil {
		log.Fatalf("error while opening a database connection: %v\n", err)
		return
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		log.Fatalf("error while getting the device store : %v\n", err)
		return
	}

	client := whatsmeow.NewClient(deviceStore, nil)
	translator := gemini.NewGeminiTranslateService(config.AppConfig.GeminiAPIKey)
	imageGenerator := gemini.NewGeminiImageGenerator(string(constants.GeminiModelImageGenerator), config.AppConfig.GeminiAPIKey)

	// Initialize the language detector with supported languages
	detector := services.NewLinguaLangDetectService(constants.SupportedLanguages)

	// Initialize transcription services using container.DB (PROVEN PATTERN)
	transcriptionState := transcription.NewStateManager(container.DB)
	transcriptionSvc := transcription.NewService(config.AppConfig.TranscribeServiceURL)

	// connect to the client and event handler
	evtHandler, err := messagehandler.NewWhatsMeowEventHandler(
		client,
		detector,
		translator,
		imageGenerator,
		transcriptionState,
		transcriptionSvc,
	)
	// ... rest of code unchanged ...
}
```

Modify `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/base.go` constructor:

```go
func NewWhatsMeowEventHandler(
	client *whatsmeow.Client,
	langDetector services.LangDetectorInterface,
	translator services.TranslateService,
	imageGenerator services.ImageGeneratorInterface,
	transcriptionState *transcription.StateManager,
	transcriptionSvc *transcription.Service,
) (*WhatsMeowEventHandler, error) {
	registry := framework.NewRegistry()

	handler := &WhatsMeowEventHandler{
		client:             client,
		langDetector:       langDetector,
		translator:         translator,
		imageGenerator:     imageGenerator,
		commandRegistry:    registry,
		transcriptionState: transcriptionState,
		transcriptionSvc:   transcriptionSvc,
	}

	if err := handler.InitializeCommands(); err != nil {
		return nil, err
	}

	return handler, nil
}
```

**Step 4: Run build to verify it compiles**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS

**Step 5: Commit**

```bash
git add internal/services/messagehandler/base.go internal/services/messagehandler/handler_adapter.go main.go
git commit -m "feat(transcription): integrate state manager and service into event handler

- Add transcriptionState and transcriptionSvc fields to event handler
- Pass services from main.go initialization
- Use existing container.DB from sqlstore (proven pattern)
- Use config.AppConfig.TranscribeServiceURL for service endpoint
- Expose via handler adapter GetStateManager/GetTranscriptionService
- Prepare for audio message processing integration"
```

---

### Task 7: Register Transcription Commands

**Files:**
- Modify: `internal/services/messagehandler/event_handler.go:82-163`

**Step 1: Write the failing test**

Run: `go test ./internal/services/messagehandler/ -v`
Expected: PASS (existing tests)

**Step 2: Verify current state**

Commands not yet registered in registry.

**Step 3: Write minimal implementation**

Modify `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/event_handler.go`:

```go
// Add to imports:
import (
	// ... existing imports ...
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/handlers/transcription"
)

// In InitializeCommands() function, after existing commands (around line 145):
func (h *WhatsMeowEventHandler) InitializeCommands() error {
	registry := h.commandRegistry

	// ... existing command registrations ...

	// Register transcription commands
	if err := registry.Register(transcription.NewTEnableCommand()); err != nil {
		return fmt.Errorf("failed to register tenable command: %w", err)
	}

	if err := registry.Register(transcription.NewTDisableCommand()); err != nil {
		return fmt.Errorf("failed to register tdisable command: %w", err)
	}

	// Apply middleware to commands that need owner permissions
	ownerCommands := []string{
		"ping", "setmodel", "settemp", "image", "meme", "randmoji",
		"haha", "download", "hibp", "tenable", "tdisable",  // Added transcription commands
	}
	// ... rest of function ...
}
```

**Step 4: Run build to verify it compiles**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS

**Step 5: Commit**

```bash
git add internal/services/messagehandler/event_handler.go
git commit -m "feat(transcription): register tenable and tdisable commands

- Add command registration in InitializeCommands
- Mark as owner-only commands in middleware
- Commands now available via /tenable and /tdisable"
```

---

### Task 8: Implement Audio Message Transcription Flow

**Files:**
- Modify: `internal/services/messagehandler/event_handler.go:19-79`
- Create: `internal/services/messagehandler/transcription.go`
- Create: `internal/services/messagehandler/transcription_test.go`

**Step 1: Write the failing test**

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/transcription_test.go`:

```go
package messagehandler

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

func TestShouldTranscribe(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	stateManager := transcription.NewStateManager(db)
	chatJID := types.JID{User: "1234567890", Server: "s.whatsapp.net"}

	// Test: disabled by default
	audioMsg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			Url: stringPtr("https://example.com/audio.ogg"),
		},
	}

	should := shouldTranscribe(audioMsg, chatJID, stateManager)
	if should {
		t.Error("expected shouldTranscribe = false when disabled")
	}

	// Enable transcription
	stateManager.Enable(context.Background(), chatJID, "he")

	should = shouldTranscribe(audioMsg, chatJID, stateManager)
	if !should {
		t.Error("expected shouldTranscribe = true when enabled")
	}

	// Test: non-audio message
	textMsg := &waProto.Message{
		Conversation: stringPtr("Hello"),
	}

	should = shouldTranscribe(textMsg, chatJID, stateManager)
	if should {
		t.Error("expected shouldTranscribe = false for non-audio message")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/messagehandler/ -v -run TestShouldTranscribe`
Expected: FAIL with "undefined: shouldTranscribe"

**Step 3: Write minimal implementation**

Create `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/transcription.go`:

```go
package messagehandler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/constants"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

func shouldTranscribe(msg *waProto.Message, chatJID types.JID, stateManager *transcription.StateManager) bool {
	// Check if it's an audio message
	if msg.GetAudioMessage() == nil {
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

	// Download audio using whatsmeow's client.Download() (PROVEN PATTERN)
	audioMsg := msg.GetAudioMessage()
	audioData, err := h.client.Download(audioMsg)
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

	// Send transcription as reply
	adapter := NewHandlerAdapter(h)
	response := fmt.Sprintf("🎤 *תמלול הודעה קולית:*\n\n%s", result.Text)
	if result.DetectedLanguage != "" && result.DetectedLanguage != language {
		response += fmt.Sprintf("\n\n🌐 Detected language: %s", result.DetectedLanguage)
	}

	adapter.SendResponse(msgInfo, response)
	return nil
}
```

Modify `/home/ilan/whatsapp-livetranslate-2.0/internal/services/messagehandler/event_handler.go`:

```go
// In handleMessage() function, BEFORE command extraction (around line 20):
func (h *WhatsMeowEventHandler) handleMessage(msg *waProto.Message, msgInfo types.MessageInfo) {
	// Check if this is an audio message that should be transcribed
	if shouldTranscribe(msg, msgInfo.Chat, h.transcriptionState) {
		if err := h.handleAudioTranscription(msg, msgInfo); err != nil {
			fmt.Printf("Audio transcription error: %v\n", err)
		}
		return // Don't process as command
	}

	// Existing command processing logic
	text := extractText(msg)
	var cmdName string
	// ... rest of existing code ...
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/messagehandler/ -v -run TestShouldTranscribe`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/messagehandler/transcription.go internal/services/messagehandler/transcription_test.go internal/services/messagehandler/event_handler.go
git commit -m "feat(transcription): implement automatic audio transcription flow

- Add shouldTranscribe check before command processing
- Download audio using whatsmeow client
- Save to temp file and upload to transcription service
- Support language-specific model selection (Ivrit for Hebrew)
- Send transcription as reply to audio message
- Silent fail if transcription service unavailable
- Add unit tests for transcription logic"
```

---

## Phase 6: Configuration and Documentation

### Task 9: Add Configuration for Transcription Service

**Files:**
- Modify: `config/config.go:11-32`
- Create: `.env.example`

**Step 1: Write the failing test**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS (config changes don't need tests)

**Step 2: Verify current state**

No TRANSCRIBE_SERVICE_URL configuration exists.

**Step 3: Write minimal implementation**

Modify `/home/ilan/whatsapp-livetranslate-2.0/config/config.go`:

```go
type config struct {
	GeminiAPIKey           string `validate:"required"`
	HIBPToken              string // HIBP API token for dark web searches
	HIBPURL                string
	TranscribeServiceURL   string // URL for transcription service
}

var (
	AppConfig = new(config)
)

func init() {
	if strings.ToLower(os.Getenv("IS_DOCKER")) != "true" {
		if err := godotenv.Load(); err != nil {
			log.Fatalf("error loading env variables: %v\n", err)
			return
		}
	}

	AppConfig.GeminiAPIKey = os.Getenv("GEMINI_API_KEY")
	AppConfig.HIBPToken = os.Getenv("HIBP_TOKEN")
	AppConfig.HIBPURL = os.Getenv("HIBP_URL")
	AppConfig.TranscribeServiceURL = os.Getenv("TRANSCRIBE_SERVICE_URL")

	// Set default transcription service URL if not provided
	if AppConfig.TranscribeServiceURL == "" {
		AppConfig.TranscribeServiceURL = "http://localhost:8009"
	}
}
```

Modify `/home/ilan/whatsapp-livetranslate-2.0/main.go` to use config:

```go
// Change line where transcriptionSvc is created:
transcriptionSvc := transcription.NewService(config.AppConfig.TranscribeServiceURL)
```

Create `/home/ilan/whatsapp-livetranslate-2.0/.env.example`:

```env
# Required
GEMINI_API_KEY=your_gemini_api_key_here

# Optional - Media Download
COOKIES_PATH=/path/to/cookies.txt
YOUTUBE_VISITOR_DATA=your_visitor_data_here

# Optional - Have I Been Pwned
HIBP_TOKEN=your_hibp_token_here
HIBP_URL=https://api.hibp.custom.endpoint

# Optional - Transcription Service (defaults to http://localhost:8009)
TRANSCRIBE_SERVICE_URL=http://localhost:8009
```

**Step 4: Run build to verify it compiles**

Run: `go build -o whatsapp-livetranslate .`
Expected: SUCCESS

**Step 5: Commit**

```bash
git add config/config.go main.go .env.example
git commit -m "feat(transcription): add configurable transcription service URL

- Add TRANSCRIBE_SERVICE_URL to config
- Default to http://localhost:8009 if not set
- Update main.go to use config value
- Add .env.example with all environment variables"
```

---

### Task 10: Update Documentation

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`

**Step 1: Update CLAUDE.md**

Modify `/home/ilan/whatsapp-livetranslate-2.0/CLAUDE.md`:

```markdown
## Required Environment Variables

The bot requires the following environment variables in a `.env` file:

- `GEMINI_API_KEY` (required) - Google Gemini API key for translation and image generation
- `COOKIES_PATH` (optional) - Path to cookies.txt for non-YouTube media downloads (Instagram, Twitter, etc.)
- `YOUTUBE_VISITOR_DATA` (optional) - YouTube visitor data for bypassing some restrictions
- `HIBP_TOKEN` (optional) - API token for Have I Been Pwned dark web search (owner-only command)
- `HIBP_URL` (optional) - Custom HIBP API endpoint URL
- `TRANSCRIBE_SERVICE_URL` (optional) - URL for live transcription service (default: http://localhost:8009)

Note: The config reads `GEMINI_API_KEY` (not `GEMINI_KEY` as mentioned in some documentation).

## Architecture Overview

### Transcription Feature

The bot supports automatic audio message transcription with per-chat enable/disable controls:

**Transcription Flow:**
1. Audio message received
2. Check if transcription enabled for chat (SQLite lookup in `transcription_settings` table)
3. If enabled:
   - Download audio using `whatsmeow.Client.Download()`
   - Save to temporary file
   - POST to transcription service REST API (`/api/transcribe/whisper-ivrit` for Hebrew)
   - Parse response and send transcription as reply
   - Clean up temp file

**Commands:**
- `/tenable [language]` - Enable transcription (default language: Hebrew)
- `/tdisable` - Disable transcription

**State Management:**
- SQLite table: `transcription_settings`
- Fields: `chat_jid`, `enabled`, `language`, `created_at`, `updated_at`
- Default language: Hebrew (`he`)

**Language Detection:**
- Uses existing `LangDetectorInterface` for smart language detection
- Falls back to configured language preference
- Model selection: Ivrit-CT2 for Hebrew, Whisper-V3-Turbo for other languages

**Transcription Service:**
- REST API at `localhost:8009` (or configured URL)
- Supports Whisper, Ivrit, and Deepgram models
- Handles file upload with multipart form data
- Returns JSON with transcription text and metadata

**Integration Point:**
- Audio messages processed in `handleMessage()` BEFORE command routing
- Silent fail if transcription service unavailable (no user notification)
- Audio messages with transcription enabled bypass command processing
```

**Step 2: Update README.md**

Modify `/home/ilan/whatsapp-livetranslate-2.0/README.md`:

Add to environment variables section:

```markdown
| Variable | Description | Required |
|----------|-------------|----------|
| `GEMINI_KEY` | Google Gemini API key for AI features | Yes |
| `YOUTUBE_VISITOR_DATA` | YouTube visitor data for bypassing some restrictions | No |
| `COOKIES_PATH` | Path to cookies.txt for non-YouTube sites (Instagram, Twitter, etc.) | No |
| `HIBP_TOKEN` | API token for Have I Been Pwned dark web search (owner only) | No |
| `TRANSCRIBE_SERVICE_URL` | URL for live transcription service (default: http://localhost:8009) | No |
```

Add to commands section:

```markdown
### Transcription Commands
- `/tenable [language]` - Enable audio transcription for this chat (owner only)
- `/tdisable` - Disable audio transcription for this chat (owner only)
- Supported languages: Hebrew (default), English, and more
```

Add to features section:

```markdown
### Core Capabilities
- **🌐 Real-time Translation**: Translate messages between 20+ languages using Google's Gemini AI
- **🎤 Audio Transcription**: Automatic transcription of voice messages with per-chat controls
- **📥 Media Downloader**: Download videos and images from YouTube, Instagram, Twitter, and more
```

**Step 3: Verify documentation**

Review both files for accuracy and completeness.

**Step 4: Commit**

```bash
git add CLAUDE.md README.md
git commit -m "docs(transcription): document audio transcription feature

- Add TRANSCRIBE_SERVICE_URL to environment variables
- Document transcription architecture and flow
- Add /tenable and /tdisable commands to README
- Explain state management and language detection
- Update feature list with audio transcription"
```

---

## Phase 7: Testing and Verification

### Task 11: Integration Testing

**Files:**
- Create: `test/integration/transcription_test.go`

**Step 1: Write integration test**

Create `/home/ilan/whatsapp-livetranslate-2.0/test/integration/transcription_test.go`:

```go
// +build integration

package integration

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
)

// This test requires the actual transcription service to be running
// Run with: go test -tags=integration ./test/integration/

func TestTranscriptionServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	service := transcription.NewService("http://localhost:8009")

	// Test health check
	resp, err := http.Get("http://localhost:8009/health")
	if err != nil {
		t.Skipf("Transcription service not available: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Transcription service not healthy: status %d", resp.StatusCode)
	}

	// Note: Actual audio transcription test would require a real audio file
	// This is a smoke test to verify service availability
	t.Log("Transcription service is available and healthy")
}

func TestStateManagerIntegration(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	stateManager := transcription.NewStateManager(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	chatJID := types.JID{User: "integration_test", Server: "s.whatsapp.net"}

	// Test full workflow
	enabled, err := stateManager.IsEnabled(ctx, chatJID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("new chat should default to disabled")
	}

	// Enable with custom language
	err = stateManager.Enable(ctx, chatJID, "en")
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	enabled, err = stateManager.IsEnabled(ctx, chatJID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if !enabled {
		t.Error("transcription should be enabled")
	}

	lang, err := stateManager.GetLanguage(ctx, chatJID)
	if err != nil {
		t.Fatalf("GetLanguage failed: %v", err)
	}
	if lang != "en" {
		t.Errorf("expected language 'en', got '%s'", lang)
	}

	// Disable
	err = stateManager.Disable(ctx, chatJID)
	if err != nil {
		t.Fatalf("Disable failed: %v", err)
	}

	enabled, err = stateManager.IsEnabled(ctx, chatJID)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("transcription should be disabled")
	}

	t.Log("State manager integration test passed")
}
```

**Step 2: Run integration tests**

Run: `go test -tags=integration ./test/integration/ -v`
Expected: PASS or SKIP (if service not running)

**Step 3: Run all unit tests**

Run: `go test ./... -v`
Expected: PASS (all tests)

**Step 4: Commit**

```bash
git add test/integration/
git commit -m "test(transcription): add integration tests

- Add integration test for transcription service health check
- Add integration test for state manager workflow
- Tests can be skipped if service unavailable
- Run with: go test -tags=integration ./test/integration/"
```

---

### Task 12: Manual Testing Checklist

**Testing Steps:**

**Step 1: Start the application**

```bash
go build -o whatsapp-livetranslate .
./whatsapp-livetranslate
```

**Step 2: Verify transcription service**

```bash
curl http://localhost:8009/health
```

Expected: `{"status":"healthy","whisper_model":"whisper-v3-turbo","model_loaded":true}`

**Step 3: Test commands via WhatsApp**

1. Send `/tenable` to the bot
   - Expected: "✅ Transcription enabled for this chat"

2. Send a voice message
   - Expected: Bot replies with transcription

3. Send `/tenable en` to the bot
   - Expected: "✅ Transcription enabled for this chat\n🌐 Language: en"

4. Send a voice message in English
   - Expected: Bot replies with English transcription

5. Send `/tdisable` to the bot
   - Expected: "🔇 Transcription disabled for this chat"

6. Send a voice message
   - Expected: No response (transcription disabled)

**Step 4: Test error handling**

1. Stop transcription service
2. Send voice message with transcription enabled
   - Expected: No response (silent fail)

3. Restart transcription service
4. Send voice message
   - Expected: Transcription works again

**Step 5: Verify database**

```bash
sqlite3 /data/auth.db "SELECT * FROM transcription_settings;"
```

Expected: See rows with chat_jid, enabled, language

**Step 6: Document results**

Create test report with screenshots and findings.

**Commit:**

```bash
git add docs/
git commit -m "docs(transcription): add manual testing checklist and results

- Document manual testing steps
- Include expected behaviors
- Add error handling verification
- Include database verification commands"
```

---

## Phase 8: Final Integration and Cleanup

### Task 13: Final Build and Deployment

**Step 1: Clean build**

```bash
go mod tidy
go build -o whatsapp-livetranslate .
```

Expected: SUCCESS with no errors

**Step 2: Test Docker build**

```bash
docker-compose build
```

Expected: SUCCESS

**Step 3: Update Docker environment**

Modify `docker-compose.yml` if needed to expose transcription service URL:

```yaml
environment:
  - TRANSCRIBE_SERVICE_URL=http://host.docker.internal:8009
```

**Step 4: Run in Docker**

```bash
docker-compose up -d
docker-compose logs -f
```

Expected: Application starts successfully

**Step 5: Verify all features**

Test translation, meme generation, download, AND transcription to ensure no regressions.

**Step 6: Commit**

```bash
git add go.mod go.sum docker-compose.yml
git commit -m "build(transcription): finalize build and deployment configuration

- Run go mod tidy to clean dependencies
- Update docker-compose with transcription service URL
- Verify Docker build succeeds
- All features tested and working"
```

---

## Completion Checklist

Before marking this feature complete, verify:

- [ ] All tests pass: `go test ./... -v`
- [ ] Build succeeds: `go build -o whatsapp-livetranslate .`
- [ ] `/tenable` command works
- [ ] `/tdisable` command works
- [ ] Audio messages are transcribed when enabled
- [ ] Audio messages are ignored when disabled
- [ ] Language parameter works (Hebrew and English tested)
- [ ] Silent fail when transcription service unavailable
- [ ] Database schema created correctly
- [ ] No regressions in existing features (translation, meme, download)
- [ ] Documentation updated (README.md and CLAUDE.md)
- [ ] `.env.example` includes new variable
- [ ] Code committed with descriptive messages
- [ ] Integration tests written and passing

---

## Notes

**Model Selection Strategy:**
- Hebrew: Use `ivrit-ct2` model (optimized for Hebrew)
- Other languages: Use `whisper-v3-turbo` (multilingual)
- Fallback: If API fails, silently skip (don't spam user)

**Error Handling Philosophy:**
- Transcription service unavailable: Silent fail (log only)
- Invalid audio format: Send error message to user
- Database errors: Send error message to user
- Download failures: Send error message to user

**Performance Considerations:**
- Temp files cleaned up immediately after transcription
- Audio downloaded to memory first, then saved to disk
- HTTP client has 5-minute timeout for long transcriptions
- SQLite queries use indexes for fast lookups

**Security:**
- Commands are owner-only (RequireOwner middleware)
- No sensitive data in transcription requests
- Temp files use secure temp directory
- Database uses existing auth.db (already secured)

**Future Enhancements (Not in This Plan):**
- Async transcription with progress updates
- Support for multiple transcription services
- Transcription history/logging
- Custom model selection per chat
- Diarization support (speaker identification)
