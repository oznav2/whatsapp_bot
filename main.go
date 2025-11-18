package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/asparkoffire/whatsapp-livetranslate-go/config"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/constants"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/gemini"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/messagehandler"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

func main() {
	ctx := context.Background()

	// Open database for both whatsmeow store and transcription
	dbPath := "file:/data/auth.db?_foreign_keys=on"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("error while opening database: %v\n", err)
		return
	}
	defer db.Close()

	// Create whatsmeow container with the existing database
	container := sqlstore.NewWithDB(db, "sqlite3", nil)

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		log.Fatalf("error while getting the device store : %v\n", err)
		return
	}

	client := whatsmeow.NewClient(deviceStore, nil)
	translator := gemini.NewGeminiTranslateService(config.AppConfig.GeminiAPIKey)
	imageGenerator := gemini.NewGeminiImageGenerator(string(constants.GeminiModelImageGenerator), config.AppConfig.GeminiAPIKey)

	// Initialize the language detector with supported languages
	detector := services.NewLinguaLangDetectService(constants.SupportedLanguages)

	// Initialize transcription services using the same database connection
	transcriptionState := transcription.NewStateManager(db)
	transcriptionSvc := transcription.NewService(config.AppConfig.TranscribeServiceURL)

	// connect to the client and event handler
	evtHandler, err := messagehandler.NewWhatsMeowEventHandler(
		client,
		detector,
		translator,
		imageGenerator,
		transcriptionState,
		transcriptionSvc,
	)
	if err != nil {
		log.Fatalf("error while setting up the event handler: %v\n", err)
		return
	}

	client.AddEventHandler(evtHandler.HandleEvents)
	fmt.Println("Server started, Listening for messages...")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
