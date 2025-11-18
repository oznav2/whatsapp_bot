package admin

import (
	"fmt"
	"strconv"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
)

type SetModelCommand struct{}

func NewSetModelCommand() *SetModelCommand {
	return &SetModelCommand{}
}

func (c *SetModelCommand) Execute(ctx *framework.Context) error {
	if len(ctx.Args) == 0 {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Error("Please specify a model ID"))
	}

	modelID := ctx.Args[0]
	if err := ctx.Handler.GetTranslator().SetModel(modelID); err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Error(fmt.Sprintf("Failed to set model: %v", err)))
	}

	return ctx.Handler.SendResponse(ctx.MessageInfo,
		framework.Success(fmt.Sprintf("Translation model set to: %s", modelID)))
}

func (c *SetModelCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:         "setmodel",
		Description:  "הגדר מודל AI לתרגום",
		Category:     "ניהול",
		Usage:        "/setmodel <model-id>",
		RequireOwner: true,
		Examples: []string{
			"/setmodel gemini-1.5-flash",
			"/setmodel gemini-2.0-flash",
		},
		Parameters: []framework.Parameter{
			{
				Name:        "model",
				Type:        framework.StringParam,
				Description: "Model ID (e.g., gemini-1.5-flash)",
				Required:    true,
			},
		},
	}
}

type GetModelCommand struct{}

func NewGetModelCommand() *GetModelCommand {
	return &GetModelCommand{}
}

func (c *GetModelCommand) Execute(ctx *framework.Context) error {
	model := ctx.Handler.GetTranslator().GetModel()
	return ctx.Handler.SendResponse(ctx.MessageInfo,
		framework.Info(fmt.Sprintf("Current translation model: %s", model)))
}

func (c *GetModelCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:         "getmodel",
		Description:  "הצג מודל תרגום נוכחי",
		Category:     "ניהול",
		Usage:        "/getmodel",
		RequireOwner: false,
	}
}

type SetTempCommand struct{}

func NewSetTempCommand() *SetTempCommand {
	return &SetTempCommand{}
}

func (c *SetTempCommand) Execute(ctx *framework.Context) error {
	if len(ctx.Args) == 0 {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Error("Please specify a temperature value between 0.0 and 1.0"))
	}

	temp, err := strconv.ParseFloat(ctx.Args[0], 64)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Error("Invalid temperature value. Please provide a number between 0.0 and 1.0"))
	}

	if err := ctx.Handler.GetTranslator().SetTemperature(temp); err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Error(fmt.Sprintf("Failed to set temperature: %v", err)))
	}

	return ctx.Handler.SendResponse(ctx.MessageInfo,
		framework.Success(fmt.Sprintf("Temperature set to: %.1f", temp)))
}

func (c *SetTempCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:         "settemp",
		Description:  "הגדר טמפרטורת AI (0.0-1.0)",
		Category:     "ניהול",
		Usage:        "/settemp <temperature>",
		RequireOwner: true,
		Examples: []string{
			"/settemp 0.7",
			"/settemp 0.3",
		},
		Parameters: []framework.Parameter{
			{
				Name:        "temperature",
				Type:        framework.FloatParam,
				Description: "Temperature value between 0.0 and 1.0",
				Required:    true,
				Validator: func(value string) error {
					temp, err := strconv.ParseFloat(value, 64)
					if err != nil {
						return err
					}
					if temp < 0.0 || temp > 1.0 {
						return fmt.Errorf("temperature must be between 0.0 and 1.0")
					}
					return nil
				},
			},
		},
	}
}

type GetTempCommand struct{}

func NewGetTempCommand() *GetTempCommand {
	return &GetTempCommand{}
}

func (c *GetTempCommand) Execute(ctx *framework.Context) error {
	temp := ctx.Handler.GetTranslator().GetTemperature()
	return ctx.Handler.SendResponse(ctx.MessageInfo,
		framework.Info(fmt.Sprintf("Current temperature: %.1f", temp)))
}

func (c *GetTempCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:         "gettemp",
		Description:  "הצג טמפרטורת AI נוכחית",
		Category:     "ניהול",
		Usage:        "/gettemp",
		RequireOwner: false,
	}
}
