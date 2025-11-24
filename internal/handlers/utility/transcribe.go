package utility

import (
	"fmt"
	"strings"

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

	// Send YouTube URL first to create preview that stays visible
	ctx.Handler.SendResponse(ctx.MessageInfo, targetURL)

	// Get video metadata from VibeGram service (silently in background)
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

	// Transcribe using Ivrit CT2 (Hebrew-optimized model) - silently, no progress messages
	request := framework.WSTranscriptionRequest{
		URL:         targetURL,
		Language:    "he",
		Model:       "ivrit-ct2",
		CaptureMode: "full",
	}

	result, err := transcriptionSvc.TranscribeViaWebSocket(ctx.Context, request, nil)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("❌ שגיאה בתמלול: %v", err))
	}

	// Final response (thumbnail already sent as separate image)
	finalResponse := fmt.Sprintf("🎬 *תמלול הושלם* (Ivrit CT2)\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s\n\n📝 *תמלול:*\n\n%s",
		videoTitle, videoDuration, result.Text)

	return ctx.Handler.SendResponse(ctx.MessageInfo, finalResponse)
}

func (c *TranscribeCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "transcribe",
		Description: "Transcribe Hebrew video using Ivrit CT2 model",
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
