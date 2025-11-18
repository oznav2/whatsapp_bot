package messagehandler

import (
	"context"
	"fmt"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/memegenerator"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/utils"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// HandlerAdapter adapts WhatsMeowEventHandler to implement framework.HandlerInterface
type HandlerAdapter struct {
	*WhatsMeowEventHandler
	mediaUploader *framework.MediaUploader
}

func NewHandlerAdapter(h *WhatsMeowEventHandler) *HandlerAdapter {
	return &HandlerAdapter{
		WhatsMeowEventHandler: h,
		mediaUploader:         framework.NewMediaUploader(&ClientAdapter{client: h.client}),
	}
}

func (a *HandlerAdapter) SendResponse(msgInfo types.MessageInfo, text string) error {
	a.WhatsMeowEventHandler.SendResponse(msgInfo, text)
	return nil
}

func (a *HandlerAdapter) SendMedia(msgInfo types.MessageInfo, mediaType framework.MediaType, data []byte, caption string) error {
	ctx := context.Background()

	switch mediaType {
	case framework.MediaImage:
		return a.mediaUploader.UploadAndSendImage(ctx, msgInfo.Chat, data, caption)
	case framework.MediaVideo:
		return a.mediaUploader.UploadAndSendVideo(ctx, msgInfo.Chat, data, caption)
	case framework.MediaDocument:
		// Use a more descriptive filename with .mp4 extension for videos
		filename := fmt.Sprintf("document_%d.mp4", msgInfo.Timestamp.Unix())
		return a.mediaUploader.UploadAndSendDocument(ctx, msgInfo.Chat, data, filename, caption)
	default:
		return fmt.Errorf("unsupported media type: %v", mediaType)
	}
}

func (a *HandlerAdapter) SendImage(msgInfo types.MessageInfo, upload framework.UploadResponse, caption string) error {
	// Create the image message
	msg := &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String("image/jpeg"),
			URL:           proto.String(upload.URL),
			DirectPath:    proto.String(upload.DirectPath),
			MediaKey:      upload.MediaKey,
			FileEncSHA256: upload.FileEncSHA256,
			FileSHA256:    upload.FileSHA256,
			FileLength:    proto.Uint64(upload.FileLength),
		},
	}

	// If the message is from us, we can't edit a media message, so we'll send a new one
	// Edit operations only work for text messages, not media
	_, err := a.client.SendMessage(context.Background(), msgInfo.Chat, msg)
	return err
}

func (a *HandlerAdapter) SendVideo(msgInfo types.MessageInfo, upload framework.UploadResponse, caption string) error {
	// Create the video message
	msg := &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String("video/mp4"),
			URL:           proto.String(upload.URL),
			DirectPath:    proto.String(upload.DirectPath),
			MediaKey:      upload.MediaKey,
			FileEncSHA256: upload.FileEncSHA256,
			FileSHA256:    upload.FileSHA256,
			FileLength:    proto.Uint64(upload.FileLength),
		},
	}

	// If the message is from us, we can't edit a media message, so we'll send a new one
	// Edit operations only work for text messages, not media
	_, err := a.client.SendMessage(context.Background(), msgInfo.Chat, msg)
	return err
}

func (a *HandlerAdapter) SendDocument(msgInfo types.MessageInfo, upload framework.UploadResponse, caption string) error {
	// Create the document message
	msg := &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			Caption:       proto.String(caption),
			FileName:      proto.String("document.mp4"), // Default to .mp4 for video documents
			Mimetype:      proto.String("video/mp4"),    // Set proper MIME type for MP4
			URL:           proto.String(upload.URL),
			DirectPath:    proto.String(upload.DirectPath),
			MediaKey:      upload.MediaKey,
			FileEncSHA256: upload.FileEncSHA256,
			FileSHA256:    upload.FileSHA256,
			FileLength:    proto.Uint64(upload.FileLength),
		},
	}

	// If the message is from us, we can't edit a media message, so we'll send a new one
	// Edit operations only work for text messages, not media
	_, err := a.client.SendMessage(context.Background(), msgInfo.Chat, msg)
	return err
}

func (a *HandlerAdapter) EditMessage(msgInfo types.MessageInfo, newText string) error {
	return a.WhatsMeowEventHandler.editMessageContent(msgInfo.Chat, msgInfo.ID, newText, nil)
}

func (a *HandlerAdapter) EditMessageWithOriginal(msgInfo types.MessageInfo, newText string, originalMsg *waProto.Message) error {
	return a.WhatsMeowEventHandler.editMessageContent(msgInfo.Chat, msgInfo.ID, newText, originalMsg)
}

func (a *HandlerAdapter) GetClient() framework.ClientInterface {
	return &ClientAdapter{client: a.client}
}

func (a *HandlerAdapter) GetTranslator() framework.TranslatorInterface {
	return &TranslatorAdapter{translator: a.translator}
}

func (a *HandlerAdapter) GetImageGenerator() framework.ImageGeneratorInterface {
	return a.imageGenerator
}

func (a *HandlerAdapter) GetMemeGenerator() framework.MemeGeneratorInterface {
	return &MemeGeneratorAdapter{generator: a.memeGenerator}
}

func (a *HandlerAdapter) GetLangDetector() framework.LangDetectorInterface {
	return &LangDetectorAdapter{detector: a.detector}
}

// ClientAdapter adapts whatsmeow.Client to implement framework.ClientInterface
type ClientAdapter struct {
	client *whatsmeow.Client
}

func (c *ClientAdapter) SendMessage(ctx context.Context, to types.JID, message *waProto.Message) (whatsmeow.SendResponse, error) {
	return c.client.SendMessage(ctx, to, message)
}

func (c *ClientAdapter) Upload(ctx context.Context, data []byte, appInfo framework.MediaType) (framework.UploadResponse, error) {
	mediaType := framework.ConvertMediaType(appInfo)
	resp, err := c.client.Upload(ctx, data, mediaType)
	if err != nil {
		return framework.UploadResponse{}, err
	}

	return framework.UploadResponse{
		URL:           resp.URL,
		DirectPath:    resp.DirectPath,
		MediaKey:      resp.MediaKey,
		FileEncSHA256: resp.FileEncSHA256,
		FileSHA256:    resp.FileSHA256,
		FileLength:    resp.FileLength,
	}, nil
}

// TranslatorAdapter adapts the TranslateService to framework.TranslatorInterface
type TranslatorAdapter struct {
	translator services.TranslateService
}

func (t *TranslatorAdapter) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	// Since the language detector returns the string representation, we need to convert back
	// The command interface expects language codes as strings for flexibility
	// For now, we'll parse the lingua.Language from the code
	source := utils.GetLangByCode(sourceLang)
	target := utils.GetLangByCode(targetLang)
	return t.translator.TranslateText(text, source, target)
}

func (t *TranslatorAdapter) SetModel(modelID string) error {
	return t.translator.SetModel(modelID)
}

func (t *TranslatorAdapter) GetModel() string {
	return t.translator.GetModel()
}

func (t *TranslatorAdapter) SetTemperature(temp float64) error {
	return t.translator.SetTemperature(temp)
}

func (t *TranslatorAdapter) GetTemperature() float64 {
	return t.translator.GetTemperature()
}

// MemeGeneratorAdapter adapts the meme generator to the interface
type MemeGeneratorAdapter struct {
	generator *memegenerator.MemeGenerator
}

func (m *MemeGeneratorAdapter) GetRandomMeme(ctx context.Context, subreddit string) (*framework.MemeResponse, error) {
	resp, err := m.generator.GetRandomMeme(ctx, subreddit)
	if err != nil {
		return nil, err
	}

	// Convert to framework.MemeResponse
	memes := make([]framework.Meme, len(resp.Memes))
	for i, meme := range resp.Memes {
		memes[i] = framework.Meme{
			Title:     meme.Title,
			URL:       meme.URL,
			Subreddit: meme.Subreddit,
		}
	}

	return &framework.MemeResponse{Memes: memes}, nil
}

func (a *HandlerAdapter) GetStateManager() *transcription.StateManager {
	return a.transcriptionState
}

func (a *HandlerAdapter) GetTranscriptionService() *transcription.Service {
	return a.transcriptionSvc
}

// LangDetectorAdapter adapts the language detector to the interface
type LangDetectorAdapter struct {
	detector services.LangDetectService
}

func (l *LangDetectorAdapter) DetectLanguage(text string) (string, error) {
	lang, ok := l.detector.DetectLanguage(text)
	if !ok {
		return "", fmt.Errorf("could not detect language")
	}
	return lang.IsoCode639_1().String(), nil
}
