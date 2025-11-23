package utility

import (
	"context"
	"fmt"
	"strings"
	"time"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/lrstanley/go-ytdlp"
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

	// Extract video metadata using yt-dlp
	dl := ytdlp.New().DumpSingleJSON().NoPlaylist()
	result, err := dl.Run(context.Background(), targetURL)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("❌ שגיאה בקבלת מידע על הווידאו: %v", err))
	}

	// Parse extracted info
	infos, err := result.GetExtractedInfo()
	if err != nil || len(infos) == 0 {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "❌ לא ניתן לחלץ מידע על הווידאו")
	}

	info := infos[0]
	videoTitle := "Unknown Video"
	if info.Title != nil && *info.Title != "" {
		videoTitle = *info.Title
	}

	videoDurationSeconds := 0
	if info.Duration != nil {
		videoDurationSeconds = int(*info.Duration)
	}
	videoDuration := formatDuration(videoDurationSeconds)

	ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("🔍 מזהה שפה...\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s", videoTitle, videoDuration))

	// Quick transcription of first 60 seconds to detect language using existing language detector
	quickRequest := framework.WSTranscriptionRequest{
		URL:         targetURL,
		Language:    "he", // Try Hebrew first
		Model:       "ivrit-ct2",
		CaptureMode: "first60",
	}

	var firstChunkText string
	quickCallback := func(msg framework.WSTranscriptionMessage) {
		if msg.Type == "transcription_chunk" && msg.Text != "" && firstChunkText == "" {
			firstChunkText = msg.Text
		}
	}

	quickResult, _ := transcriptionSvc.TranscribeViaWebSocket(ctx.Context, quickRequest, quickCallback)

	// Use existing language detector from bot
	detectedLangCode := "he" // Default to Hebrew
	languageName := "Hebrew"
	model := "ivrit-ct2"

	if quickResult != nil && strings.TrimSpace(quickResult.Text) != "" {
		// Use the bot's existing language detection on the transcribed text
		langDetector := ctx.Handler.GetLangDetector()
		detectedLang, err := langDetector.DetectLanguage(quickResult.Text)
		if err == nil && detectedLang != "" {
			detectedLangCode = detectedLang
			// If not Hebrew, use Deepgram
			if detectedLangCode != "he" && detectedLangCode != "iw" {
				model = "deepgram"
				languageName = detectedLangCode
			}
		}
	} else {
		// Hebrew transcription failed, try Deepgram
		model = "deepgram"
		detectedLangCode = "en"
		languageName = "English"
	}

	verboseMsg := fmt.Sprintf("🎬 מתמלל עכשיו \"%s\"\n⏱️ משך: %s\n🔤 שפה מזוהה: %s",
		videoTitle, videoDuration, languageName)
	ctx.Handler.SendResponse(ctx.MessageInfo, verboseMsg)

	// Full transcription with detected language and model
	fullRequest := framework.WSTranscriptionRequest{
		URL:         targetURL,
		Language:    detectedLangCode,
		Model:       model,
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

	fullResult, err := transcriptionSvc.TranscribeViaWebSocket(ctx.Context, fullRequest, progressCallback)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("❌ שגיאה בתמלול: %v", err))
	}

	finalResponse := fmt.Sprintf("🎬 *תמלול הושלם*\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s\n🔤 שפה: %s\n\n📝 *תמלול:*\n\n%s",
		videoTitle, videoDuration, languageName, fullResult.Text)

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
