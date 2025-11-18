package fun

import (
	"context"
	"fmt"
	"strings"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
)

type ImageCommand struct{}

func NewImageCommand() *ImageCommand {
	return &ImageCommand{}
}

func (c *ImageCommand) Execute(ctx *framework.Context) error {
	if len(ctx.Args) == 0 {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Error("Please provide a prompt for image generation"))
	}

	prompt := ctx.RawArgs

	// Send processing message
	ctx.Handler.SendResponse(ctx.MessageInfo,
		framework.Processing(fmt.Sprintf("Generating image: %s", prompt)))

	// Generate image
	imageBytes, err := ctx.Handler.GetImageGenerator().GenerateImage(context.Background(), prompt)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Error(fmt.Sprintf("Failed to generate image: %v", err)))
	}

	// Send generated image
	return ctx.Handler.SendMedia(ctx.MessageInfo, framework.MediaImage, imageBytes, prompt)
}

func (c *ImageCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:         "image",
		Aliases:      []string{"img", "generate"},
		Description:  "צור תמונה באמצעות AI",
		Category:     "בידור",
		Usage:        "/image <prompt>",
		RequireOwner: true,
		Examples: []string{
			"/image a beautiful sunset over mountains",
			"/image cyberpunk city at night",
			"/image cute robot painting a picture",
		},
		Parameters: []framework.Parameter{
			{
				Name:        "prompt",
				Type:        framework.StringParam,
				Description: "Description of the image to generate",
				Required:    true,
				Validator: func(value string) error {
					if len(strings.TrimSpace(value)) < 3 {
						return fmt.Errorf("prompt must be at least 3 characters")
					}
					return nil
				},
			},
		},
	}
}
