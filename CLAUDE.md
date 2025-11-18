# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
```bash
# Build the application
go build -o whatsapp-livetranslate .

# Run directly
./whatsapp-livetranslate

# Run with Go
go run main.go

# Using Docker
docker-compose up -d
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/cmdframework/
go test ./internal/handlers/admin/
```

### Code Formatting
```bash
# Format all Go files
gofmt -w .
```

### Dependencies
```bash
# Download dependencies
go mod download

# Update dependencies
go mod tidy
```

## Required Environment Variables

The bot requires the following environment variables in a `.env` file:

- `GEMINI_API_KEY` (required) - Google Gemini API key for translation and image generation
- `COOKIES_PATH` (optional) - Path to cookies.txt for non-YouTube media downloads (Instagram, Twitter, etc.)
- `YOUTUBE_VISITOR_DATA` (optional) - YouTube visitor data for bypassing some restrictions
- `HIBP_TOKEN` (optional) - API token for Have I Been Pwned dark web search (owner-only command)
- `HIBP_URL` (optional) - Custom HIBP API endpoint URL
- `TRANSCRIBE_SERVICE_URL` (optional) - URL for audio transcription service (defaults to http://localhost:8009)

Note: The config reads `GEMINI_API_KEY` (not `GEMINI_KEY` as mentioned in some documentation).

## Architecture Overview

### Command Framework Pattern

The bot uses an interface-based command framework where all commands implement:

```go
type Command interface {
    Execute(ctx *Context) error
    Metadata() *Metadata
}
```

**Command Context** provides access to:
- Message information (`ctx.MessageInfo`, `ctx.Message`)
- Parsed arguments (`ctx.Args`, `ctx.RawArgs`)
- Service interfaces via `ctx.Handler`:
  - `GetTranslator()` - Gemini translation service
  - `GetImageGenerator()` - Gemini image generation
  - `GetMemeGenerator()` - Reddit meme fetching
  - `GetLangDetector()` - Lingua language detection
  - `GetClient()` - WhatsApp client wrapper

### Command Registration Flow

1. Commands are organized by category in `internal/handlers/`:
   - `admin/` - Model/temperature configuration
   - `fun/` - Image generation, memes, emoji spam
   - `translation/` - Language translation commands
   - `transcription/` - Audio transcription controls (tenable, tdisable)
   - `utility/` - Download, ping, sed, HIBP

2. All commands are registered in `internal/services/messagehandler/event_handler.go` in the `InitializeCommands()` function.

3. **Important**: Translation commands for all supported languages are auto-registered via `translation.RegisterTranslationCommands(registry)` rather than individually.

4. Owner-only commands use middleware wrapping:
   ```go
   wrappedCmd := framework.WithMiddleware(cmd, framework.RequireOwner())
   registry.Register(wrappedCmd)
   ```

### Service Layer

The application follows a dependency injection pattern:

- `main.go` creates service instances (Gemini translator, image generator, language detector)
- Services are passed to `WhatsMeowEventHandler`
- Commands access services through the `Context.Handler` interface
- This allows for testing commands with mock services

### WhatsApp Integration

- Uses `whatsmeow` library for WhatsApp Multi-Device support
- Session data stored in SQLite database at `/data/auth.db`
- First run requires QR code scan for device pairing
- Event handler pattern processes incoming messages and routes to commands

## Adding New Commands

When adding a new command:

1. Create the command file in the appropriate `internal/handlers/` subdirectory
2. Implement both `Execute(ctx *Context)` and `Metadata()` methods
3. Register in `InitializeCommands()` in `event_handler.go`:
   ```go
   if err := registry.Register(yourcategory.NewYourCommand()); err != nil {
       return fmt.Errorf("failed to register yourcommand: %w", err)
   }
   ```
4. If owner-only, add command name to the `ownerCommands` slice in `InitializeCommands()`

See `docs/ADDING_COMMANDS.md` for detailed examples.

## Special Command Handling

- **Sed command** (`s/find/replace/`) has special parsing in `handleMessage()` - it doesn't use the standard `/` prefix
- **Translation commands** are dynamically registered for all language codes in `constants.SupportedLanguages`
- **Language detection** falls back to 2-character language codes when a command is not found in the registry

## Owner Detection

Owner permissions are determined by the bot owner's WhatsApp JID (set during first connection). Owner-only commands are wrapped with `framework.RequireOwner()` middleware which checks the sender's JID before execution.

## Media Download Architecture

The download command uses `yt-dlp` (via `go-ytdlp` wrapper) for media extraction from:
- YouTube (uses `YOUTUBE_VISITOR_DATA` if provided)
- Instagram, Twitter/X, TikTok (uses `COOKIES_PATH` for authenticated content)
- Other supported platforms

Downloads have a 1-minute rate limit to prevent abuse.

## Audio Transcription Architecture

The bot supports automatic transcription of audio messages with per-chat enable/disable controls.

### Components

**State Manager** (`internal/services/transcription/state_manager.go`):
- SQLite-based persistence for per-chat transcription settings
- Shares the same database connection as whatsmeow (`/data/auth.db`)
- Schema: `transcription_settings` table with `chat_jid`, `enabled`, `language` columns
- Methods: `IsEnabled()`, `Enable()`, `Disable()`, `GetLanguage()`

**Transcription Service** (`internal/services/transcription/service.go`):
- HTTP client for external transcription service (VibeGram v5.0 API)
- Multipart file upload with 5-minute timeout
- Endpoint selection based on language:
  - Hebrew: `/api/transcribe/whisper-ivrit` with `ivrit-ct2` model
  - Other languages: `/api/transcribe/whisper` with `whisper-v3-turbo` model

**Commands** (`internal/handlers/transcription/`):
- `/tenable [language]` - Enable transcription (owner-only, defaults to Hebrew)
- `/tdisable` - Disable transcription (owner-only)

### Message Flow

1. **Audio message received** → `handleMessage()` in `event_handler.go`
2. **Check if transcription enabled** → `shouldTranscribe()` checks message type and state
3. **Download audio** → `client.Download(ctx, audioMsg)` downloads audio data
4. **Save to temp file** → Creates temporary `.ogg` file
5. **Transcribe** → `transcriptionSvc.TranscribeAudio()` sends to API
6. **Send response** → Posts transcription as reply to original message
7. **Return early** → Audio messages are NOT processed as commands

### Integration Points

- Event handler integration: `internal/services/messagehandler/transcription.go`
- Audio message detection: `shouldTranscribe()` checks for `AudioMessage` and enabled state
- Silent failures: If transcription service is unavailable, errors are logged but not shown to users
- Model selection: Automatic based on configured language (ivrit-ct2 for Hebrew, whisper-v3-turbo otherwise)

### Database Sharing

The transcription state manager shares the database connection with whatsmeow's session store:
```go
db, err := sql.Open("sqlite3", "file:/data/auth.db?_foreign_keys=on")
container := sqlstore.NewWithDB(db, "sqlite3", nil)
transcriptionState := transcription.NewStateManager(db)
```

This ensures both session data and transcription settings are persisted in a single database file.
