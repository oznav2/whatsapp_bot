package messagehandler

import (
	"context"
	"fmt"
	"os"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/memegenerator"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

type WhatsMeowEventHandler struct {
	client             *whatsmeow.Client
	detector           services.LangDetectService
	translator         services.TranslateService
	imageGenerator     services.ImageGenerator
	memeGenerator      *memegenerator.MemeGenerator
	commandRegistry    *framework.Registry
	transcriptionState *transcription.StateManager
	transcriptionSvc   *transcription.Service
}

func NewWhatsMeowEventHandler(
	client *whatsmeow.Client,
	detector services.LangDetectService,
	translator services.TranslateService,
	imageGenerator services.ImageGenerator,
	transcriptionState *transcription.StateManager,
	transcriptionSvc *transcription.Service,
) (*WhatsMeowEventHandler, error) {
	handler := &WhatsMeowEventHandler{
		client:             client,
		detector:           detector,
		translator:         translator,
		imageGenerator:     imageGenerator,
		memeGenerator:      memegenerator.NewMemeGenerator(),
		commandRegistry:    framework.NewRegistry(),
		transcriptionState: transcriptionState,
		transcriptionSvc:   transcriptionSvc,
	}

	// Initialize all commands
	if err := handler.InitializeCommands(); err != nil {
		return nil, err
	}

	if handler.client.Store.ID == nil {
		if err := handler.setupQRLogin(); err != nil {
			return nil, err
		}
	} else {
		if err := client.Connect(); err != nil {
			return nil, err
		}
	}
	return handler, nil
}

func (h *WhatsMeowEventHandler) HandleEvents(evt any) {
	switch v := evt.(type) {
	case *events.Message:
		h.handleMessage(v.Message, v.Info)
	}
}

func (h *WhatsMeowEventHandler) setupQRLogin() error {
	qrChan, _ := h.client.GetQRChannel(context.Background())
	err := h.client.Connect()
	if err != nil {
		return err
	}

	for evt := range qrChan {
		if evt.Event == "code" {
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
		} else {
			fmt.Println("Login event:", evt.Event)
		}
	}
	return nil
}
