package utility

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/lrstanley/go-ytdlp"
)

type DownloadCommand struct {
	mu               sync.Mutex
	isDownloading    bool
	lastDownloadTime time.Time
}

var downloadInstance = &DownloadCommand{}

func NewDownloadCommand() *DownloadCommand {
	return downloadInstance
}

func (c *DownloadCommand) Execute(ctx *framework.Context) error {
	if len(ctx.Args) == 0 {
		return ctx.Handler.SendResponse(ctx.MessageInfo, framework.Error("Please provide a URL to download"))
	}

	// Check rate limiting
	c.mu.Lock()

	// Check if download is already in progress
	if c.isDownloading {
		c.mu.Unlock()
		return ctx.Handler.SendResponse(ctx.MessageInfo, framework.Warning("⏳ A download is already in progress. Please wait for it to complete."))
	}

	// Check cooldown period (1 minute)
	if !c.lastDownloadTime.IsZero() {
		timeSinceLastDownload := time.Since(c.lastDownloadTime)
		if timeSinceLastDownload < 5*time.Second {
			remainingTime := 5*time.Second - timeSinceLastDownload
			c.mu.Unlock()
			return ctx.Handler.SendResponse(ctx.MessageInfo, framework.Warning(fmt.Sprintf("⏱️ Please wait %d seconds before downloading again.", int(remainingTime.Seconds()))))
		}
	}

	// Mark as downloading
	c.isDownloading = true
	c.mu.Unlock()

	// Ensure we mark as not downloading when done
	defer func() {
		c.mu.Lock()
		c.isDownloading = false
		c.lastDownloadTime = time.Now()
		c.mu.Unlock()
		fmt.Printf("[DOWNLOAD] Download completed, cooldown started\n")
	}()

	url := ctx.Args[0]
	fmt.Printf("[DOWNLOAD] Starting download for URL: %s\n", url)

	// Send initial processing message that we'll edit later
	initialMsg := framework.Processing("Starting download...")
	err := ctx.Handler.SendResponse(ctx.MessageInfo, initialMsg)
	if err != nil {
		fmt.Printf("[DOWNLOAD] Failed to send initial message: %v\n", err)
	}

	// Create temporary directory for downloads
	tempDir, err := os.MkdirTemp("", "whatsapp-download-*")
	if err != nil {
		fmt.Printf("[DOWNLOAD] Failed to create temp directory: %v\n", err)
		// Reset download state on early error
		c.mu.Lock()
		c.isDownloading = false
		c.mu.Unlock()
		errorMsg := framework.Error("Failed to create temp directory")
		ctx.Handler.EditMessage(ctx.MessageInfo, errorMsg)
		return nil
	}
	defer func() {
		fmt.Printf("[DOWNLOAD] Cleaning up temp directory: %s\n", tempDir)
		os.RemoveAll(tempDir)
	}()
	fmt.Printf("[DOWNLOAD] Created temp directory: %s\n", tempDir)

	// Update message to show downloading
	processingMsg := framework.Processing("Downloading media...")
	ctx.Handler.EditMessage(ctx.MessageInfo, processingMsg)

	// Configure output template
	outputTemplate := filepath.Join(tempDir, "download.%(ext)s")
	fmt.Printf("[DOWNLOAD] Output template: %s\n", outputTemplate)

	// Initialize ytdlp with options
	dl := ytdlp.New().
		Format("best[height<=720]/best"). // Better format selection
		NoPlaylist().                     // Download only single video, not playlist
		RestrictFilenames().              // Safe filenames
		Output(outputTemplate).
		NoCheckCertificates(). // Skip certificate verification
		Verbose()              // Enable verbose logging

	// Check if URL is YouTube
	isYouTube := strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")

	// For all other sites, try to use cookies if available
	dl = dl.CookiesFromBrowser("firefox:/root/profile/")
	dl = dl.AddHeaders(fmt.Sprintf("User-Agent:%s", os.Getenv("USER_AGENT")))

	// Download the media
	fmt.Printf("[DOWNLOAD] Running yt-dlp...\n")
	result, err := dl.Run(context.Background(), url)
	if err != nil {
		fmt.Printf("[DOWNLOAD] yt-dlp failed: %v\n", err)

		// Check for common errors
		errStr := err.Error()

		var errorMsg string
		if isYouTube && (strings.Contains(errStr, "Sign in to confirm") || strings.Contains(errStr, "age")) {
			errorMsg = framework.Error("❌ This video is age-restricted.\n\nTo download it, set YOUTUBE_VISITOR_DATA environment variable.\nSee README for instructions.")
		} else if strings.Contains(errStr, "This content isn't available") {
			errorMsg = framework.Error("❌ Content unavailable. Video may be private, deleted, or region-blocked.")
		} else if !isYouTube && (strings.Contains(errStr, "login") || strings.Contains(errStr, "private") || strings.Contains(errStr, "authenticate")) {
			errorMsg = framework.Error("❌ This content requires authentication.\n\nExport cookies from your browser and set COOKIES_PATH.\nSee README for instructions.")
		} else {
			errorMsg = framework.Error(fmt.Sprintf("Download failed: %v", err))
		}

		ctx.Handler.EditMessage(ctx.MessageInfo, errorMsg)
		return nil
	}

	if result != nil {
		fmt.Printf("[DOWNLOAD] yt-dlp exit code: %d\n", result.ExitCode)
		if result.Stdout != "" {
			fmt.Printf("[DOWNLOAD] yt-dlp stdout:\n%s\n", result.Stdout)
		}
		if result.Stderr != "" {
			fmt.Printf("[DOWNLOAD] yt-dlp stderr:\n%s\n", result.Stderr)
		}
	}

	// Wait a moment for file to be fully written
	time.Sleep(500 * time.Millisecond)

	// Update message to show processing
	processingMsg = framework.Processing("Processing downloaded file...")
	ctx.Handler.EditMessage(ctx.MessageInfo, processingMsg)

	// Find the downloaded file
	files, err := filepath.Glob(filepath.Join(tempDir, "download.*"))
	if err != nil {
		fmt.Printf("[DOWNLOAD] Glob error: %v\n", err)
		errorMsg := framework.Error("Error finding downloaded file")
		ctx.Handler.EditMessage(ctx.MessageInfo, errorMsg)
		return nil
	}

	fmt.Printf("[DOWNLOAD] Found %d files in temp directory\n", len(files))
	for i, f := range files {
		info, _ := os.Stat(f)
		if info != nil {
			fmt.Printf("[DOWNLOAD] File %d: %s (size: %d bytes)\n", i+1, f, info.Size())
		}
	}

	if len(files) == 0 {
		// Try alternative patterns
		alternativePatterns := []string{
			filepath.Join(tempDir, "*.mp4"),
			filepath.Join(tempDir, "*.webm"),
			filepath.Join(tempDir, "*.mkv"),
			filepath.Join(tempDir, "*"),
		}

		for _, pattern := range alternativePatterns {
			altFiles, _ := filepath.Glob(pattern)
			if len(altFiles) > 0 {
				files = altFiles
				fmt.Printf("[DOWNLOAD] Found files with pattern %s\n", pattern)
				break
			}
		}

		if len(files) == 0 {
			// List all files in temp directory for debugging
			allFiles, _ := os.ReadDir(tempDir)
			fmt.Printf("[DOWNLOAD] All files in temp directory:\n")
			for _, f := range allFiles {
				fmt.Printf("[DOWNLOAD]   - %s\n", f.Name())
			}
			errorMsg := framework.Error("Downloaded file not found")
			ctx.Handler.EditMessage(ctx.MessageInfo, errorMsg)
			return nil
		}
	}
	outputFile := files[0]
	fmt.Printf("[DOWNLOAD] Selected file: %s\n", outputFile)

	// Get file info
	fileInfo, err := os.Stat(outputFile)
	if err != nil {
		fmt.Printf("[DOWNLOAD] Failed to stat file: %v\n", err)
		errorMsg := framework.Error("Failed to access downloaded file")
		ctx.Handler.EditMessage(ctx.MessageInfo, errorMsg)
		return nil
	}
	fmt.Printf("[DOWNLOAD] File size: %d bytes (%.2f MB)\n", fileInfo.Size(), float64(fileInfo.Size())/(1024*1024))

	// Read the file
	data, err := os.ReadFile(outputFile)
	if err != nil {
		fmt.Printf("[DOWNLOAD] Failed to read file: %v\n", err)
		errorMsg := framework.Error("Failed to read downloaded file")
		ctx.Handler.EditMessage(ctx.MessageInfo, errorMsg)
		return nil
	}
	fmt.Printf("[DOWNLOAD] Read %d bytes from file\n", len(data))

	// Check if file is actually empty
	if len(data) == 0 {
		fmt.Printf("[DOWNLOAD] ERROR: File is empty!\n")
		errorMsg := framework.Error("Downloaded file is empty")
		ctx.Handler.EditMessage(ctx.MessageInfo, errorMsg)
		return nil
	}

	// Check file size (WhatsApp has limits)
	const maxSize = 16 * 1024 * 1024 // 16MB limit for WhatsApp
	if len(data) > maxSize {
		fmt.Printf("[DOWNLOAD] File too large: %d bytes\n", len(data))
		// For large files, we'll send as document instead
		fmt.Printf("[DOWNLOAD] File exceeds 16MB limit, sending as document...\n")

		// Prepare caption with size info
		caption := fmt.Sprintf("📥 הורד מ: %s\n\n📎 File is %.1f MB (exceeds 16MB limit for media)", url, float64(len(data))/(1024*1024))

		// Upload as document with proper filename to preserve extension
		filename := filepath.Base(outputFile)
		if filename == "" || filename == "." {
			filename = fmt.Sprintf("video_%d.mp4", time.Now().Unix())
		}

		uploader := framework.NewMediaUploader(ctx.Handler.GetClient())
		resp, err := uploader.UploadDocument(ctx.Context, data, filename)
		if err != nil {
			fmt.Printf("[DOWNLOAD] Document upload failed: %v\n", err)
			errorMsg := framework.Error(fmt.Sprintf("Failed to upload document: %v", err))
			ctx.Handler.EditMessage(ctx.MessageInfo, errorMsg)
			return nil
		}

		// Send document with caption
		err = ctx.Handler.SendDocument(ctx.MessageInfo, resp, caption)
		if err != nil {
			fmt.Printf("[DOWNLOAD] Failed to send document message: %v\n", err)
			return err
		}
		fmt.Printf("[DOWNLOAD] Large file sent as document successfully\n")
		return nil
	}

	// Prepare caption
	caption := fmt.Sprintf("📥 הורד מ: %s", url)

	// Send based on type using the media uploader's UploadAndSend methods
	uploader := framework.NewMediaUploader(ctx.Handler.GetClient())
	extension := strings.ToLower(filepath.Ext(outputFile))
	fmt.Printf("[DOWNLOAD] File extension: %s\n", extension)

	if extension == ".mp4" || extension == ".webm" || extension == ".mkv" || extension == ".avi" {
		fmt.Printf("[DOWNLOAD] Uploading and sending as video...\n")
		// Upload and send as video
		err := uploader.UploadAndSendVideo(ctx.Context, ctx.MessageInfo.Chat, data, caption)
		if err != nil {
			fmt.Printf("[DOWNLOAD] Video upload/send failed: %v\n", err)
			errorMsg := framework.Error(fmt.Sprintf("Failed to upload/send video: %v", err))
			ctx.Handler.EditMessage(ctx.MessageInfo, errorMsg)
			return nil
		}
		fmt.Printf("[DOWNLOAD] Video sent successfully\n")
		return nil
	} else {
		// Upload and send as image or document
		if isImageFile(outputFile) {
			err := uploader.UploadAndSendImage(ctx.Context, ctx.MessageInfo.Chat, data, caption)
			if err != nil {
				errorMsg := framework.Error(fmt.Sprintf("Failed to upload/send image: %v", err))
				ctx.Handler.EditMessage(ctx.MessageInfo, errorMsg)
				return nil
			}
			return nil
		} else {
			// Send as document if not recognized media type
			// Preserve the original filename to maintain file type recognition
			filename := filepath.Base(outputFile)
			if filename == "" || filename == "." {
				// If we still can't determine the filename, use a generic one with proper extension
				ext := filepath.Ext(outputFile)
				if ext == "" {
					ext = ".mp4" // Default to MP4 for video files
				}
				filename = fmt.Sprintf("media_%d%s", time.Now().Unix(), ext)
			}
			err := uploader.UploadAndSendDocument(ctx.Context, ctx.MessageInfo.Chat, data, filename, caption)
			if err != nil {
				errorMsg := framework.Error(fmt.Sprintf("Failed to upload/send document: %v", err))
				ctx.Handler.EditMessage(ctx.MessageInfo, errorMsg)
				return nil
			}
			return nil
		}
	}
}

func (c *DownloadCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "download",
		Aliases:     []string{"dl", "ytdl"},
		Description: "הורד מדיה מפלטפורמות שונות",
		Category:    "כלי שירות",
		Usage:       "/download <url>",
		Examples: []string{
			"/download https://www.youtube.com/watch?v=...",
			"/dl https://www.instagram.com/p/...",
			"/dl https://twitter.com/user/status/...",
		},
	}
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// downloadWithHTTP is a fallback for direct image/video URLs
func downloadWithHTTP(url string) ([]byte, string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("bad status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Get filename from URL or content disposition
	filename := filepath.Base(url)
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if idx := strings.Index(cd, "filename="); idx != -1 {
			filename = strings.Trim(cd[idx+9:], "\"")
		}
	}

	return data, filename, nil
}
