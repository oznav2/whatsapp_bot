package utility

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
)

// SedCommand represents the /sed command
type SedCommand struct{}

// NewSedCommand creates a new instance of SedCommand
func NewSedCommand() *SedCommand {
	return &SedCommand{}
}

// Name returns the name of the command
func (c *SedCommand) Name() string {
	return "s"
}

// Description returns the description of the command
func (c *SedCommand) Description() string {
	return "Applies a sed-like substitution to a quoted message. Usage: s/pattern/replacement/flags or /s s/pattern/replacement/flags. Flags: g - global, i - ignore case, c - strikethrough original text"
}

// Execute executes the sed command
func (c *SedCommand) Execute(ctx *framework.Context) error {
	if ctx.Message.ExtendedTextMessage == nil || ctx.Message.ExtendedTextMessage.ContextInfo == nil || ctx.Message.ExtendedTextMessage.ContextInfo.QuotedMessage == nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "Please quote a message to use the sed command.")
	}

	quotedText := ctx.Message.ExtendedTextMessage.ContextInfo.QuotedMessage.GetConversation()
	if quotedText == "" {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "Could not extract text from the quoted message.")
	}
	log.Printf("Sed Command: Quoted text: %s", quotedText)

	sedExpression := ctx.RawArgs
	log.Printf("Sed Command: Expression: %s", sedExpression)
	// Expected format: s/pattern/replacement/flags
	// Using a simple regex to parse the components. This assumes '/' as delimiter.
	// We need to escape the first '/' to match literally, and then capture everything between the next two '/'
	// and then everything after the last '/'
	// Regex: ^s/(.*?)/(.*?)/(.*)?$
	// Group 1: pattern
	// Group 2: replacement
	// Group 3: flags (optional)

	// Using a more robust regex that allows for escaping the delimiter if needed,
	// but for simplicity, we'll stick to the basic '/' delimiter as per standard sed usage.
	// A more robust parser would handle escaped delimiters within pattern/replacement.
	// For this exercise, we'll assume the format s/pattern/replacement/flags
	re := regexp.MustCompile(`^s/(.*?)/(.*?)/(.*)?$`)
	matches := re.FindStringSubmatch(sedExpression)

	if len(matches) < 3 {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "Invalid sed expression. Usage: s/pattern/replacement/flags")
	}

	pattern := matches[1]
	replacement := matches[2]
	flags := matches[3]
	log.Printf("Sed Command: Pattern: '%s', Replacement: '%s', Flags: '%s'", pattern, replacement, flags)

	// Handle flags
	reFlags := ""
	global := false
	crossOut := false
	caseInsensitive := false

	if strings.Contains(flags, "g") {
		global = true
	}
	if strings.Contains(flags, "i") {
		reFlags += "(?i)" // Add case-insensitive flag to regex
		caseInsensitive = true
	}
	if strings.Contains(flags, "c") {
		crossOut = true
	}
	log.Printf("Sed Command Flags: Global: %t, Case-Insensitive: %t, Cross-out: %t", global, caseInsensitive, crossOut)

	// Compile the pattern regex
	compiledPattern, err := regexp.Compile(reFlags + pattern)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("Invalid regex pattern: %v", err))
	}
	log.Printf("Sed Command: Compiled Regex: %s", compiledPattern.String())

	var editedText string
	if global {
		log.Println("Sed Command: Global replacement")
		if crossOut {
			log.Println("Sed Command: Cross-out enabled")
			editedText = compiledPattern.ReplaceAllStringFunc(quotedText, func(match string) string {
				return fmt.Sprintf("~%s~ %s", match, replacement)
			})
		} else {
			editedText = compiledPattern.ReplaceAllString(quotedText, replacement)
		}
	} else {
		log.Println("Sed Command: Single replacement")
		// Replace only the first occurrence
		firstMatchIndex := compiledPattern.FindStringIndex(quotedText)
		if firstMatchIndex != nil {
			matchedString := quotedText[firstMatchIndex[0]:firstMatchIndex[1]]
			if crossOut {
				log.Println("Sed Command: Cross-out enabled")
				editedText = quotedText[:firstMatchIndex[0]] + fmt.Sprintf("~%s~ %s", matchedString, replacement) + quotedText[firstMatchIndex[1]:]
			} else {
				editedText = quotedText[:firstMatchIndex[0]] + compiledPattern.ReplaceAllString(matchedString, replacement) + quotedText[firstMatchIndex[1]:]
			}
		} else {
			log.Println("Sed Command: No match found")
			// No match found, text remains unchanged
			editedText = quotedText
		}
	}
	log.Printf("Sed Command: Output: %s", editedText)

	if editedText == quotedText {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "No changes were made. Pattern not found or expression invalid.")
	}

	return ctx.Handler.SendResponse(ctx.MessageInfo, editedText)
}

func (c *SedCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "s",
		Description: "מבצע החלפת טקסט בהודעה מצוטטת. דגלים: g - גלובלי, i - התעלם מגדול/קטן, c - קו חוצה על הטקסט המקורי",
		Category:    "כלי שירות",
		Usage:       "s/pattern/replacement/flags or /s s/pattern/replacement/flags",
	}
}
