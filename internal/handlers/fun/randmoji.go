package fun

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/constants"
)

var emojiRunning int32 = 0

type RandmojiCommand struct{}

func NewRandmojiCommand() *RandmojiCommand {
	return &RandmojiCommand{}
}

func (c *RandmojiCommand) Execute(ctx *framework.Context) error {
	// Only allow one instance to run at a time
	if !atomic.CompareAndSwapInt32(&emojiRunning, 0, 1) {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Warning("Randmoji is already running"))
	}
	defer atomic.StoreInt32(&emojiRunning, 0)

	duration := 10 // default duration
	if len(ctx.Args) > 0 {
		if d, err := strconv.Atoi(ctx.Args[0]); err == nil && d > 0 && d <= 10 {
			duration = d
		}
	}

	// Run emoji animation in background
	go func() {
		for i := 0; i < duration; i++ {
			for j := 0; j < 3; j++ {
				time.Sleep(500 * time.Millisecond)
				emoji := getRandomEmoji()
				ctx.Handler.EditMessage(ctx.MessageInfo, fmt.Sprintf("```%s```", emoji))
			}
		}
	}()

	return nil
}

func (c *RandmojiCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:         "randmoji",
		Description:  "הצג אימוג'ים אקראיים",
		Category:     "בידור",
		Usage:        "/randmoji [duration]",
		RequireOwner: true,
		Examples: []string{
			"/randmoji",
			"/randmoji 5",
		},
		Parameters: []framework.Parameter{
			{
				Name:        "duration",
				Type:        framework.IntParam,
				Description: "Number of cycles (1-10)",
				Required:    false,
				Default:     10,
				Validator: func(value string) error {
					d, err := strconv.Atoi(value)
					if err != nil {
						return err
					}
					if d < 1 || d > 10 {
						return fmt.Errorf("duration must be between 1 and 10")
					}
					return nil
				},
			},
		},
	}
}

func getRandomEmoji() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return constants.Emojis[rng.Intn(len(constants.Emojis))]
}
