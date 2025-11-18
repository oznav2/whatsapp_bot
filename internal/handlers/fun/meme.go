package fun

import (
	"context"
	"fmt"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
)

type MemeCommand struct{}

func NewMemeCommand() *MemeCommand {
	return &MemeCommand{}
}

func (c *MemeCommand) Execute(ctx *framework.Context) error {
	var subreddit string
	if len(ctx.Args) > 0 {
		subreddit = ctx.Args[0]
	}

	// Show fetching status
	statusMsg := "🔍 Fetching random meme"
	if subreddit != "" {
		statusMsg = fmt.Sprintf("🔍 Fetching meme from r/%s", subreddit)
	}
	ctx.Handler.SendResponse(ctx.MessageInfo, statusMsg)

	// Fetch meme
	memeResp, err := ctx.Handler.GetMemeGenerator().GetRandomMeme(context.Background(), subreddit)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Error(fmt.Sprintf("Failed to fetch meme: %v", err)))
	}

	if len(memeResp.Memes) == 0 {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Warning("No memes found"))
	}

	meme := memeResp.Memes[0]

	// Download meme image
	imageData, err := framework.DownloadMedia(meme.URL)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Error(fmt.Sprintf("Failed to download meme: %v", err)))
	}

	// Send meme
	caption := fmt.Sprintf("r/%s: %s", meme.Subreddit, meme.Title)
	return ctx.Handler.SendMedia(ctx.MessageInfo, framework.MediaImage, imageData, caption)
}

func (c *MemeCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:         "meme",
		Description:  "קבל מם אקראי",
		Category:     "בידור",
		Usage:        "/meme [subreddit]",
		RequireOwner: true,
		Examples: []string{
			"/meme",
			"/meme dankmemes",
			"/meme wholesomememes",
		},
		Parameters: []framework.Parameter{
			{
				Name:        "subreddit",
				Type:        framework.StringParam,
				Description: "Specific subreddit to get meme from",
				Required:    false,
			},
		},
	}
}
