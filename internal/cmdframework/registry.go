package cmdframework

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Registry struct {
	mu         sync.RWMutex
	commands   map[string]Command
	aliases    map[string]string
	categories map[string][]string
}

func NewRegistry() *Registry {
	return &Registry{
		commands:   make(map[string]Command),
		aliases:    make(map[string]string),
		categories: make(map[string][]string),
	}
}

func (r *Registry) Register(cmd Command) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	meta := cmd.Metadata()
	if meta.Name == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	name := strings.ToLower(meta.Name)

	if _, exists := r.commands[name]; exists {
		return fmt.Errorf("command %s already registered", name)
	}

	if _, exists := r.aliases[name]; exists {
		return fmt.Errorf("command name %s conflicts with existing alias", name)
	}

	r.commands[name] = cmd

	for _, alias := range meta.Aliases {
		alias = strings.ToLower(alias)
		if _, exists := r.commands[alias]; exists {
			return fmt.Errorf("alias %s conflicts with existing command", alias)
		}
		if _, exists := r.aliases[alias]; exists {
			return fmt.Errorf("alias %s already registered", alias)
		}
		r.aliases[alias] = name
	}

	if meta.Category != "" {
		r.categories[meta.Category] = append(r.categories[meta.Category], name)
	}

	return nil
}

func (r *Registry) Get(name string) (Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	name = strings.ToLower(name)

	if cmd, exists := r.commands[name]; exists {
		return cmd, true
	}

	if actualName, exists := r.aliases[name]; exists {
		return r.commands[actualName], true
	}

	return nil, false
}

func (r *Registry) GetAll() map[string]Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]Command)
	for name, cmd := range r.commands {
		result[name] = cmd
	}
	return result
}

func (r *Registry) GetByCategory(category string) []Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Command
	if names, exists := r.categories[category]; exists {
		for _, name := range names {
			if cmd, exists := r.commands[name]; exists {
				result = append(result, cmd)
			}
		}
	}
	return result
}

func (r *Registry) GetCategories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	categories := make([]string, 0, len(r.categories))
	for cat := range r.categories {
		categories = append(categories, cat)
	}
	sort.Strings(categories)
	return categories
}

func (r *Registry) GenerateHelp() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("📋 *פקודות זמינות*\n\n")

	categories := r.GetCategories()

	uncategorized := make([]string, 0)
	categorizedCommands := make(map[string]bool)

	for _, cat := range categories {
		if len(r.categories[cat]) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("*%s*\n", cat))

		commands := r.categories[cat]
		sort.Strings(commands)

		for _, cmdName := range commands {
			cmd := r.commands[cmdName]
			meta := cmd.Metadata()
			if !meta.Hidden {
				sb.WriteString(fmt.Sprintf("• */%s*", meta.Name))
				if meta.Description != "" {
					sb.WriteString(fmt.Sprintf(" - %s", meta.Description))
				}
				sb.WriteString("\n")
				categorizedCommands[cmdName] = true
			}
		}
		sb.WriteString("\n")
	}

	for name, cmd := range r.commands {
		if !categorizedCommands[name] {
			meta := cmd.Metadata()
			if !meta.Hidden {
				uncategorized = append(uncategorized, name)
			}
		}
	}

	if len(uncategorized) > 0 {
		sb.WriteString("*פקודות אחרות*\n")
		sort.Strings(uncategorized)
		for _, name := range uncategorized {
			cmd := r.commands[name]
			meta := cmd.Metadata()
			sb.WriteString(fmt.Sprintf("• */%s*", meta.Name))
			if meta.Description != "" {
				sb.WriteString(fmt.Sprintf(" - %s", meta.Description))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (r *Registry) GenerateCommandHelp(cmdName string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmd, exists := r.Get(cmdName)
	if !exists {
		return fmt.Sprintf("פקודה '%s' לא נמצאה", cmdName)
	}

	meta := cmd.Metadata()
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*פקודה:* */%s*\n", meta.Name))

	if len(meta.Aliases) > 0 {
		aliases := make([]string, len(meta.Aliases))
		for i, alias := range meta.Aliases {
			aliases[i] = fmt.Sprintf("*/%s*", alias)
		}
		sb.WriteString(fmt.Sprintf("*כינויים:* %s\n", strings.Join(aliases, ", ")))
	}

	if meta.Description != "" {
		sb.WriteString(fmt.Sprintf("*תיאור:* %s\n", meta.Description))
	}

	if meta.Usage != "" {
		sb.WriteString(fmt.Sprintf("*שימוש:* `%s`\n", meta.Usage))
	}

	if len(meta.Parameters) > 0 {
		sb.WriteString("\n*פרמטרים:*\n")
		for _, param := range meta.Parameters {
			sb.WriteString(fmt.Sprintf("• `%s`", param.Name))
			if param.Required {
				sb.WriteString(" *(חובה)*")
			}
			if param.Description != "" {
				sb.WriteString(fmt.Sprintf(" - %s", param.Description))
			}
			sb.WriteString("\n")
		}
	}

	if len(meta.Examples) > 0 {
		sb.WriteString("\n*דוגמאות:*\n")
		for _, example := range meta.Examples {
			sb.WriteString(fmt.Sprintf("• `%s`\n", example))
		}
	}

	if meta.RequireOwner {
		sb.WriteString("\n⚠️ *פקודה זו דורשת הרשאות בעלים*")
	}

	return sb.String()
}
