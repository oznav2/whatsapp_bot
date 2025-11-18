package utility

import (
	"fmt"
	"time"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
)

type PingCommand struct {
	startTime time.Time
}

func NewPingCommand() *PingCommand {
	return &PingCommand{}
}

func (c *PingCommand) Execute(ctx *framework.Context) error {
	c.startTime = time.Now()
	latency := time.Since(c.startTime)
	response := fmt.Sprintf("🏓 Pong! Latency: %s", latency)
	return ctx.Handler.SendResponse(ctx.MessageInfo, response)
}

func (c *PingCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:         "ping",
		Description:  "בדוק תגובתיות הבוט",
		Category:     "כלי שירות",
		Usage:        "/ping",
		RequireOwner: true,
	}
}
