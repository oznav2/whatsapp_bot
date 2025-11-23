package utility

import (
	"context"
	"fmt"
	"strings"
	"time"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
)

type TranscribeCommand struct{}

func NewTranscribeCommand() *TranscribeCommand {
	return &TranscribeCommand{}
}

func (c *TranscribeCommand) Execute(ctx *framework.Context) error {
	if len(ctx.Args) == 0 {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "❌ אנא ספק URL של וידאו\n\nדוגמה:\n/transcribe https://youtube.com/watch?v=...")
	}

	targetURL := ctx.Args[0]

	if !strings.HasPrefix(targetURL, "https:") && !strings.HasPrefix(targetURL, "http:") {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "❌ URL לא תקין")
	}

	transcriptionSvc := ctx.Handler.GetTranscriptionService()
	if transcriptionSvc == nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "❌ שירות התמלול לא זמין")
	}

	// Send single initial status (fewer messages = less likely to be hidden)
	ctx.Handler.SendResponse(ctx.MessageInfo, "📊 מקבל מידע ומזהה שפה...")

	// Get video metadata from VibeGram service
	metadata, err := transcriptionSvc.GetVideoMetadata(ctx.Context, targetURL)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("❌ שגיאה בקבלת מידע על הווידאו: %v", err))
	}

	videoTitle := metadata.Title
	if videoTitle == "" {
		videoTitle = "Unknown Video"
	}

	videoDuration := metadata.DurationFormatted
	if videoDuration == "" {
		videoDuration = formatDuration(metadata.DurationSeconds)
	}

	thumbnailURL := metadata.Thumbnail

	// Language detection: Try Hebrew first with first60 mode
	// Add timeout to prevent hanging on Hebrew videos
	detectCtx, cancel := context.WithTimeout(ctx.Context, 90*time.Second)
	defer cancel()

	quickRequest := framework.WSTranscriptionRequest{
		URL:         targetURL,
		Language:    "he",
		Model:       "ivrit-ct2",
		CaptureMode: "first60",
	}

	quickResult, _ := transcriptionSvc.TranscribeViaWebSocket(detectCtx, quickRequest, nil)

	// Detect language using Lingua on transcribed text
	selectedModel := "ivrit-ct2"
	selectedLanguage := "he"
	languageDisplayName := "Hebrew"

	if quickResult != nil && strings.TrimSpace(quickResult.Text) != "" {
		langDetector := ctx.Handler.GetLangDetector()
		detectedLang, err := langDetector.DetectLanguage(quickResult.Text)
		if err == nil && detectedLang != "" {
			selectedLanguage = detectedLang
			// If not Hebrew, use whisper-v3-turbo (VibeGram's multilingual model)
			if detectedLang != "he" && detectedLang != "iw" {
				selectedModel = "whisper-v3-turbo"
				languageDisplayName = detectedLang
			}
		}
	} else {
		// Hebrew transcription failed, assume non-Hebrew
		selectedModel = "whisper-v3-turbo"
		selectedLanguage = "en"
		languageDisplayName = "English"
	}

	// Display video info with detected language (single consolidated message)
	ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("🎬 מתמלל: \"%s\"\n⏱️ משך: %s\n🔤 שפה: %s\n🖼️ %s",
		videoTitle, videoDuration, languageDisplayName, thumbnailURL))

	// Full transcription with detected language and model
	fullRequest := framework.WSTranscriptionRequest{
		URL:         targetURL,
		Language:    selectedLanguage,
		Model:       selectedModel,
		CaptureMode: "full",
	}

	lastPercent := 0.0
	progressCallback := func(msg framework.WSTranscriptionMessage) {
		switch msg.Type {
		case "download_progress":
			// Only show 50% to minimize messages
			if msg.Percent >= 50.0 && lastPercent < 50.0 {
				lastPercent = msg.Percent
				ctx.Handler.SendResponse(ctx.MessageInfo, "📥 מוריד... 50%")
			}
		case "transcription_chunk":
			// Collect silently
		}
	}

	result, err := transcriptionSvc.TranscribeViaWebSocket(ctx.Context, fullRequest, progressCallback)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("❌ שגיאה בתמלול: %v", err))
	}

	// Use detected language from final result, fallback to selected language
	finalLanguage := result.DetectedLanguage
	if finalLanguage == "" {
		finalLanguage = languageDisplayName
	}

	// Final response with thumbnail URL visible
	finalResponse := fmt.Sprintf("🎬 *תמלול הושלם*\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s\n🔤 שפה: %s\n🖼️ %s\n\n📝 *תמלול:*\n\n%s",
		videoTitle, videoDuration, finalLanguage, thumbnailURL, result.Text)

	return ctx.Handler.SendResponse(ctx.MessageInfo, finalResponse)
}

func (c *TranscribeCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "transcribe",
		Description: "Transcribe video from URL",
		Usage:       "/transcribe [url]",
		Examples:    []string{"/transcribe https://youtube.com/watch?v=xyz"},
		Category:    "Utility",
	}
}

func formatDuration(seconds int) string {
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
