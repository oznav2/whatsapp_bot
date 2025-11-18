package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/constants"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services"
	gemini "github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/gemini/schemas"
	"github.com/pemistahl/lingua-go"
)

type geminiTranslateService struct {
	client             *http.Client
	modelID            constants.GeminiModel
	generateContentAPI string
	geminiAPIKey       string
	maxRetries         int
	initialBackoff     time.Duration
	maxBackoff         time.Duration
	temperature        float64
}

func NewGeminiTranslateService(geminiAPIKey string) services.TranslateService {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	return &geminiTranslateService{
		client:             client,
		modelID:            constants.Gemini20FlashLite,
		generateContentAPI: "streamGenerateContent",
		geminiAPIKey:       geminiAPIKey,
		maxRetries:         3,
		initialBackoff:     500 * time.Millisecond,
		maxBackoff:         5 * time.Second,
		temperature:        constants.DefaultTemperature,
	}
}

func (g *geminiTranslateService) TranslateText(text string, sourceLang lingua.Language, targetLang lingua.Language) (string, error) {
	var lastErr error
	backoff := g.initialBackoff

	for attempt := 1; attempt <= g.maxRetries; attempt++ {
		// Try the translation
		result, err := g.executeTranslation(text, sourceLang, targetLang)

		// If successful, return the result
		if err == nil {
			if attempt > 1 {
				fmt.Printf("Translation succeeded on attempt %d\n", attempt)
			}
			return result, nil
		}

		// If this was the last attempt, return the error
		lastErr = err
		if attempt == g.maxRetries {
			break
		}

		// Calculate next backoff with exponential increase (but don't exceed max)
		if backoff < g.maxBackoff {
			backoff = backoff * 2
			if backoff > g.maxBackoff {
				backoff = g.maxBackoff
			}
		}

		fmt.Printf("Translation attempt %d failed: %v\n", attempt, err)
		fmt.Printf("Retrying in %v...\n", backoff)
		time.Sleep(backoff)
	}

	return "", fmt.Errorf("translation failed after %d attempts: %w", g.maxRetries, lastErr)
}

func (g *geminiTranslateService) executeTranslation(text string, sourceLang lingua.Language, targetLang lingua.Language) (string, error) {
	fmt.Printf("[GEMINI] Translating from %s to %s: %s\n", sourceLang.String(), targetLang.String(), text)
	
	// Construct the Gemini URL with the model ID, API method, and API key.
	geminiUrl := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:%s?key=%s", g.modelID, g.generateContentAPI, g.geminiAPIKey)

	// Create the payload using the NewGeminiLLMInferenceRequest function.
	payload := gemini.NewGeminiLLMInferenceRequest(text, sourceLang, targetLang)

	// Marshal the payload into JSON.
	b, err := json.MarshalIndent(payload, "", "    ")
	if err != nil {
		fmt.Printf("[GEMINI] JSON marshal error: %v\n", err)
		return "", err
	}

	// Create the HTTP request.
	req, err := http.NewRequest(http.MethodPost, geminiUrl, bytes.NewReader(b))
	if err != nil {
		fmt.Printf("[GEMINI] HTTP request creation error: %v\n", err)
		return "", err
	}

	// Set the Content-Type header.
	req.Header.Set("Content-Type", "application/json")

	// Execute the request.
	res, err := g.client.Do(req)
	if err != nil {
		fmt.Printf("[GEMINI] HTTP request execution error: %v\n", err)
		return "", err
	}
	defer res.Body.Close()

	// Check if the response status code is between 200 and 299.
	if res.StatusCode < 200 || res.StatusCode > 299 {
		bodyBytes, _ := io.ReadAll(res.Body)
		fmt.Printf("[GEMINI] API error (status %d): %s\n", res.StatusCode, string(bodyBytes))
		return "", fmt.Errorf("error: received non-success status code %d: %s", res.StatusCode, string(bodyBytes))
	}

	var texts []string

	var responses []gemini.GeminiResponses
	if err := json.NewDecoder(res.Body).Decode(&responses); err != nil {
		fmt.Printf("[GEMINI] Response decode error: %v\n", err)
		return "", err
	}

	for _, response := range responses {
		for _, candidate := range response.Candidates {
			for _, part := range candidate.Content.Parts {
				texts = append(texts, part.Text)
			}
		}
	}

	final := strings.Join(texts, "")
	fmt.Printf("[GEMINI] Raw response: %s\n", final)
	
	var output gemini.OutputText
	if err := json.Unmarshal([]byte(final), &output); err != nil {
		fmt.Printf("[GEMINI] Output unmarshal error: %v\n", err)
		return "", err
	}

	fmt.Printf("[GEMINI] Translation successful: %s\n", output.Output)
	// Return the translated output.
	return output.Output, nil
}

func (g *geminiTranslateService) SetModel(modelID string) error {
	model := constants.GeminiModel(modelID)
	if _, ok := constants.ValidGeminiModels[model]; !ok {
		// Build the list of supported models dynamically
		var supportedModels []string
		for modelID := range constants.ValidGeminiModels {
			supportedModels = append(supportedModels, string(modelID))
		}

		return fmt.Errorf("invalid model ID: %s. Supported models are: %s",
			model,
			strings.Join(supportedModels, ", "),
		)
	}

	g.modelID = model
	return nil
}

func (g *geminiTranslateService) GetModel() string {
	return string(g.modelID)
}

func (g *geminiTranslateService) SetTemperature(temp float64) error {
	if temp < constants.MinTemperature || temp > constants.MaxTemperature {
		return fmt.Errorf("temperature must be between %.1f and %.1f", constants.MinTemperature, constants.MaxTemperature)
	}
	g.temperature = temp
	return nil
}

func (g *geminiTranslateService) GetTemperature() float64 {
	return g.temperature
}
