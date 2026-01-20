package parser

import (
	"regexp"
	"strings"
)

type Keymap struct {
	Name   string  `json:"name"`
	Layers []Layer `json:"layers"`
	Layout *Layout `json:"layout,omitempty"` // Physical layout for self-contained keymap files
}

type Layer struct {
	Name        string            `json:"name"`
	Keys        []string          `json:"keys"`        // Flat array of key labels, indexed by position
	CustomNames map[string]string `json:"customNames"` // Custom names: key index (as string) -> custom label
}

// ParseKeymap parses a ZMK keymap file content and returns a Keymap structure
func ParseKeymap(content string, name string) (*Keymap, error) {
	keymap := &Keymap{
		Name:   name,
		Layers: []Layer{},
	}

	// Find all ZMK_LAYER definitions using balanced parentheses matching
	prefix := "ZMK_LAYER"
	idx := 0
	for {
		start := strings.Index(content[idx:], prefix)
		if start == -1 {
			break
		}
		start += idx

		// Find the opening paren
		parenStart := strings.Index(content[start:], "(")
		if parenStart == -1 {
			break
		}
		parenStart += start

		// Find matching closing paren using balance counting
		parenEnd := findMatchingParen(content, parenStart)
		if parenEnd == -1 {
			idx = parenStart + 1
			continue
		}

		// Extract content between parens
		innerContent := content[parenStart+1 : parenEnd]

		// Split into layer name and keys (first comma separates them)
		commaIdx := strings.Index(innerContent, ",")
		if commaIdx == -1 {
			idx = parenEnd + 1
			continue
		}

		layerName := strings.TrimSpace(innerContent[:commaIdx])
		keysContent := innerContent[commaIdx+1:]

		layer := Layer{
			Name:        formatLayerName(layerName),
			Keys:        parseKeysFlat(keysContent),
			CustomNames: make(map[string]string),
		}
		keymap.Layers = append(keymap.Layers, layer)

		idx = parenEnd + 1
	}

	return keymap, nil
}

// findMatchingParen finds the index of the closing paren that matches the opening paren at startIdx
func findMatchingParen(s string, startIdx int) int {
	if startIdx >= len(s) || s[startIdx] != '(' {
		return -1
	}

	depth := 1
	for i := startIdx + 1; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// formatLayerName converts snake_case to Title Case
func formatLayerName(name string) string {
	name = strings.TrimSuffix(name, "_layer")
	words := strings.Split(name, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

// parseKeysFlat extracts key bindings as a flat array
func parseKeysFlat(content string) []string {
	// Clean up content
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\t", " ")

	return tokenize(content)
}

// tokenize splits the content into ZMK binding tokens
func tokenize(content string) []string {
	var tokens []string
	content = strings.TrimSpace(content)

	// Match ZMK bindings: &name or &name(args) or &name ARG or &name ARG1 ARG2
	bindingRegex := regexp.MustCompile(`&(\w+)(?:\s*\([^)]*\))?`)

	// Find all bindings with their positions
	matches := bindingRegex.FindAllStringSubmatchIndex(content, -1)

	for i, match := range matches {
		start := match[0]
		var end int
		if i < len(matches)-1 {
			end = matches[i+1][0]
		} else {
			end = len(content)
		}

		binding := strings.TrimSpace(content[start:end])
		label := convertBinding(binding)
		tokens = append(tokens, label)
	}

	return tokens
}

// convertBinding converts a ZMK binding to a readable label
func convertBinding(binding string) string {
	binding = strings.TrimSpace(binding)

	// Handle common patterns
	if binding == "&trans" {
		return "▽"
	}
	if binding == "&none" {
		return ""
	}

	// &kp KEY - basic keypress
	if strings.HasPrefix(binding, "&kp ") {
		key := strings.TrimPrefix(binding, "&kp ")
		key = strings.Fields(key)[0]
		return formatKey(key)
	}

	// &lt LAYER KEY - layer tap
	if strings.HasPrefix(binding, "&lt ") {
		parts := strings.Fields(binding)
		if len(parts) >= 3 {
			layer := parts[1]
			key := parts[2]
			return formatKey(key) + "/" + formatLayerShort(layer)
		}
	}

	// &mo LAYER - momentary layer
	if strings.HasPrefix(binding, "&mo ") {
		parts := strings.Fields(binding)
		if len(parts) >= 2 {
			return "[" + formatLayerShort(parts[1]) + "]"
		}
	}

	// &hrm MOD KEY - home row mod
	if strings.HasPrefix(binding, "&hrm ") {
		parts := strings.Fields(binding)
		if len(parts) >= 3 {
			return formatKey(parts[2])
		}
	}

	// &bt BT_* - bluetooth
	if strings.HasPrefix(binding, "&bt ") {
		parts := strings.Fields(binding)
		if len(parts) >= 2 {
			cmd := parts[1]
			if strings.HasPrefix(cmd, "BT_SEL") && len(parts) >= 3 {
				return "BT" + parts[2]
			}
			if cmd == "BT_CLR" {
				return "BT CLR"
			}
		}
		return "BT"
	}

	// &bootloader
	if strings.HasPrefix(binding, "&bootloader") {
		return "BOOT"
	}

	// &caps_word
	if strings.HasPrefix(binding, "&caps_word") {
		return "CAPS"
	}

	// &leader
	if strings.HasPrefix(binding, "&leader") {
		return "LDR"
	}

	// &bl BL_* - backlight
	if strings.HasPrefix(binding, "&bl ") {
		return "BL"
	}

	// &studio_unlock
	if strings.HasPrefix(binding, "&studio") {
		return "STUDIO"
	}

	// Default: extract the binding name
	re := regexp.MustCompile(`&(\w+)`)
	match := re.FindStringSubmatch(binding)
	if len(match) >= 2 {
		return strings.ToUpper(match[1][:min(4, len(match[1]))])
	}

	return "?"
}

// formatKey formats a ZMK key code to a readable label
func formatKey(key string) string {
	keyMap := map[string]string{
		"SPACE":       "SPC",
		"ENTER":       "ENT",
		"RETURN":      "RET",
		"BACKSPACE":   "BSPC",
		"BSPC":        "BSPC",
		"TAB":         "TAB",
		"ESC":         "ESC",
		"ESCAPE":      "ESC",
		"DELETE":      "DEL",
		"DEL":         "DEL",
		"INSERT":      "INS",
		"HOME":        "HOME",
		"END":         "END",
		"PAGE_UP":     "PGUP",
		"PG_UP":       "PGUP",
		"PAGE_DOWN":   "PGDN",
		"PG_DN":       "PGDN",
		"UP":          "↑",
		"DOWN":        "↓",
		"LEFT":        "←",
		"RIGHT":       "→",
		"LSHIFT":      "SHFT",
		"RSHIFT":      "SHFT",
		"LSHFT":       "SHFT",
		"LEFT_SHIFT":  "SHFT",
		"LCTRL":       "CTRL",
		"RCTRL":       "CTRL",
		"LEFT_CONTROL": "CTRL",
		"LALT":        "ALT",
		"RALT":        "ALT",
		"LGUI":        "GUI",
		"RGUI":        "GUI",
		"GRAVE":       "`",
		"MINUS":       "-",
		"EQUAL":       "=",
		"LBKT":        "[",
		"RBKT":        "]",
		"LBRC":        "{",
		"RBRC":        "}",
		"BSLH":        "\\",
		"SEMI":        ";",
		"SQT":         "'",
		"COMMA":       ",",
		"DOT":         ".",
		"SLASH":       "/",
		"FSLH":        "/",
		"CAPS":        "CAPS",
		"CAPSLOCK":    "CAPS",
		"PSCRN":       "PSCR",
		"SLCK":        "SLCK",
		"PAUSE_BREAK": "PAUS",
		"LPAR":        "(",
		"RPAR":        ")",
		"C_VOL_UP":    "V+",
		"C_VOL_DN":    "V-",
		"C_MUTE":      "MUTE",
		"C_PLAY_PAUSE": "▶⏸",
		"C_NEXT":      "⏭",
		"C_PREV":      "⏮",
	}

	// Check for modifiers like LS(), LC(), LA(), LG()
	modRegex := regexp.MustCompile(`^(L[SACG]|R[SACG]|LC|LA|LG|LS)\((.+)\)$`)
	if match := modRegex.FindStringSubmatch(key); len(match) == 3 {
		inner := formatKey(match[2])
		mod := match[1]
		switch mod {
		case "LS", "RS":
			return "S-" + inner
		case "LC", "RC":
			return "C-" + inner
		case "LA", "RA":
			return "A-" + inner
		case "LG", "RG":
			return "G-" + inner
		}
	}

	// Number keys
	if strings.HasPrefix(key, "N") && len(key) == 2 {
		return key[1:]
	}

	// Function keys
	if strings.HasPrefix(key, "F") && len(key) <= 3 {
		return key
	}

	if mapped, ok := keyMap[key]; ok {
		return mapped
	}

	// Return as-is if short enough, otherwise truncate
	if len(key) <= 4 {
		return key
	}
	return key[:4]
}

// formatLayerShort creates a short label for layer references
func formatLayerShort(layer string) string {
	switch strings.ToUpper(layer) {
	case "DEFAULT", "0":
		return "D"
	case "LOWER", "1":
		return "L"
	case "RAISE", "2":
		return "R"
	case "FN", "3":
		return "F"
	case "SYSTEM", "4":
		return "S"
	default:
		if len(layer) > 0 {
			return strings.ToUpper(layer[:1])
		}
		return "?"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
