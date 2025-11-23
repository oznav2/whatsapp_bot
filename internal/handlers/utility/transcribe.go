package utility

import (
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

	// Send initial status
	ctx.Handler.SendResponse(ctx.MessageInfo, "📊 מקבל מידע על הווידאו...")

	// Get video metadata from deepgram service (working endpoint from logs)
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

	// Display video info and thumbnail
	thumbnailURL := metadata.Thumbnail
	ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("🎬 מתמלל עכשיו \"%s\"\n⏱️ משך: %s\n🖼️ %s", videoTitle, videoDuration, thumbnailURL))

	// Language detection: Try Hebrew first with first60 mode
	ctx.Handler.SendResponse(ctx.MessageInfo, "🔍 מזהה שפה...")

	quickRequest := framework.WSTranscriptionRequest{
		URL:         targetURL,
		Language:    "he",
		Model:       "ivrit-ct2",
		CaptureMode: "first60",
	}

	quickResult, _ := transcriptionSvc.TranscribeViaWebSocket(ctx.Context, quickRequest, nil)

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

	ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("🔤 שפה מזוהה: %s\n📥 מתחיל תמלול מלא...", languageDisplayName))

	// Full transcription with detected language and model
	fullRequest := framework.WSTranscriptionRequest{
		URL:         targetURL,
		Language:    selectedLanguage,
		Model:       selectedModel,
		CaptureMode: "full",
	}

	lastPercent := 0.0
	lastUpdate := time.Now()
	progressCallback := func(msg framework.WSTranscriptionMessage) {
		switch msg.Type {
		case "download_progress":
			now := time.Now()
			if msg.Percent-lastPercent >= 25.0 || msg.Percent >= 99.0 || now.Sub(lastUpdate) > 2*time.Second {
				lastPercent = msg.Percent
				lastUpdate = now
				progressMsg := fmt.Sprintf("📥 הורדה: %.0f%%", msg.Percent)
				ctx.Handler.SendResponse(ctx.MessageInfo, progressMsg)
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
