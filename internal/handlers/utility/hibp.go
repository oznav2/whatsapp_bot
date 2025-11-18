package utility

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/asparkoffire/whatsapp-livetranslate-go/config"
	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
)

type HIBPCommand struct{}

func NewHIBPCommand() *HIBPCommand {
	return &HIBPCommand{}
}

func (c *HIBPCommand) Execute(ctx *framework.Context) error {
	// Check if HIBP token is configured
	if config.AppConfig.HIBPToken == "" {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Error("HIBP API token not configured. Please set HIBP_TOKEN in environment variables."))
	}

	// Check if user provided a search query
	if len(ctx.Args) == 0 {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Error("Please provide a search term. Usage: /hibp <search_term>"))
	}

	// Get the search term
	searchTerm := strings.Join(ctx.Args, " ")

	// Send processing message
	err := ctx.Handler.SendResponse(ctx.MessageInfo, framework.Processing("Searching for data breaches..."))
	if err != nil {
		return err
	}

	// Perform the HIBP API search
	results, err := c.searchHIBP(searchTerm)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo,
			framework.Error(fmt.Sprintf("Error searching HIBP: %v", err)))
	}

	// Format and send results
	response := c.formatResults(searchTerm, results)
	return ctx.Handler.SendResponse(ctx.MessageInfo, response)
}

func (c *HIBPCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "hibp",
		Description: "חפש דליפות מידע וחשבונות פגועים באמצעות leakosintapi.com",
		Category:    "כלי שירות",
		Usage:       "/hibp <phone_or_identifier>",
		Examples: []string{
			"/hibp +912345678901",
			"/hibp example@gmail.com",
		},
		RequireOwner: true,
		Parameters: []framework.Parameter{
			{
				Name:        "phone_or_identifier",
				Type:        framework.StringParam,
				Description: "Phone number or identifier to search for in data breach databases",
				Required:    true,
			},
		},
	}
}

// HIBP API Response structure based on the provided example
type HIBPResult struct {
	List             map[string]DatabaseInfo `json:"List"`
	NumOfDatabase    int                     `json:"NumOfDatabase"`
	NumOfResults     int                     `json:"NumOfResults"`
	FreeRequestsLeft int                     `json:"free_requests_left"`
	Price            float64                 `json:"price"`
	SearchTime       float64                 `json:"search time"`
}

type DatabaseInfo struct {
	Data         []DataRecord `json:"Data"`
	InfoLeak     string       `json:"InfoLeak"`
	NumOfResults int          `json:"NumOfResults"`
}

type DataRecord struct {
	// Common fields
	Address        string `json:"Address,omitempty"`
	Address2       string `json:"Address2,omitempty"`
	Phone          string `json:"Phone,omitempty"`
	Phone2         string `json:"Phone2,omitempty"`
	Phone3         string `json:"Phone3,omitempty"`
	Phone4         string `json:"Phone4,omitempty"`
	Phone5         string `json:"Phone5,omitempty"`
	FullName       string `json:"FullName,omitempty"`
	FirstName      string `json:"FirstName,omitempty"`
	FatherName     string `json:"FatherName,omitempty"`
	Email          string `json:"Email,omitempty"`
	DocNumber      string `json:"DocNumber,omitempty"`
	Gender         string `json:"Gender,omitempty"`
	BDay           string `json:"BDay,omitempty"`
	RegDate        string `json:"RegDate,omitempty"`
	LastActive     string `json:"LastActive,omitempty"`
	OS             string `json:"OS,omitempty"`
	PasswordMD5    string `json:"Password(MD5),omitempty"`
	Region         string `json:"Region,omitempty"`
	City           string `json:"City,omitempty"`
	Country        string `json:"Country,omitempty"`
	State          string `json:"State,omitempty"`
	District       string `json:"District,omitempty"`
	Age            string `json:"Age,omitempty"`
	CompanyName    string `json:"CompanyName,omitempty"`
	JobTitle       string `json:"JobTitle,omitempty"`
	Site           string `json:"Site,omitempty"`
	Tags           string `json:"Tags,omitempty"`
	MobileOperator string `json:"MobileOperator,omitempty"`
	IndianState    string `json:"IndianState,omitempty"`
}

// searchHIBP performs a search using the HIBP API
func (c *HIBPCommand) searchHIBP(term string) (*HIBPResult, error) {
	// Prepare API request to the documented endpoint
	// Using leakosintapi.com as per the curl example
	apiURL := "https://leakosintapi.com/"

	// Prepare JSON data as per the curl example
	data := map[string]any{
		"token":   config.AppConfig.HIBPToken,
		"request": term,
		"limit":   100,
		"lang":    "en",
		"type":    "json",
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create POST request
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers - using application/json as per the curl example
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "WhatsApp-LiveTranslate-Bot/1.0")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result HIBPResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &result, nil
}

// formatResults formats the HIBP search results for display, focusing on hitekgroop.in data
func (c *HIBPCommand) formatResults(_ string, results *HIBPResult) string {
	if results == nil {
		return framework.Error("No results received from HIBP API")
	}

	var buffer bytes.Buffer

	// Header
	// Check if we have data for hitekgroop.in specifically
	if hitekData, exists := results.List["HiTeckGroop.in"]; exists && hitekData.NumOfResults > 0 {
		buffer.WriteString("🎯 *Jackpot!!!*\n")
		buffer.WriteString(fmt.Sprintf("📄 Records found: %d\n\n", hitekData.NumOfResults))

		// Add information about the leak
		buffer.WriteString(fmt.Sprintf("⚠️ *Information about the leak:*\n%s\n\n", hitekData.InfoLeak))

		// Display data records
		for _, record := range hitekData.Data {
			// Display relevant fields for hitekgroop.in
			if record.FullName != "" {
				buffer.WriteString(fmt.Sprintf("👤 Full Name: %s\n", record.FullName))
			}
			if record.FatherName != "" {
				buffer.WriteString(fmt.Sprintf("👨 Father's Name: %s\n", record.FatherName))
			}
			if record.DocNumber != "" {
				buffer.WriteString(fmt.Sprintf("🆔  Document Number: %s\n", record.DocNumber))
			}
			if record.Address != "" {
				buffer.WriteString(fmt.Sprintf("🏠 Address: %s\n", record.Address))
			}
			if record.Address2 != "" {
				buffer.WriteString(fmt.Sprintf("🏘️ Address 2: %s\n", record.Address2))
			}
			if record.Region != "" {
				buffer.WriteString(fmt.Sprintf("📍 Region: %s\n", record.Region))
			}

			// Display all phone numbers if they exist
			phones := []string{}
			if record.Phone != "" {
				phones = append(phones, record.Phone)
			}
			if record.Phone2 != "" {
				phones = append(phones, record.Phone2)
			}
			if record.Phone3 != "" {
				phones = append(phones, record.Phone3)
			}
			if record.Phone4 != "" {
				phones = append(phones, record.Phone4)
			}
			if record.Phone5 != "" {
				phones = append(phones, record.Phone5)
			}

			if len(phones) > 0 {
				buffer.WriteString(fmt.Sprintf("📱 Phones: %s\n", strings.Join(phones, ", ")))
			}

			buffer.WriteString("\n")
		}
	} else {
		buffer.WriteString("❌ No data found for that phone number.\n\n")
	}

	return buffer.String()
}
