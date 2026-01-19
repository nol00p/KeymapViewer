package parser

import (
	"encoding/json"
)

// PhysicalKey represents a single key's physical position and size
type PhysicalKey struct {
	X      float64 `json:"x"`      // X position in key units
	Y      float64 `json:"y"`      // Y position in key units
	W      float64 `json:"w"`      // Width in key units (default 1)
	H      float64 `json:"h"`      // Height in key units (default 1)
	R      float64 `json:"r"`      // Rotation angle in degrees
	RX     float64 `json:"rx"`     // Rotation center X
	RY     float64 `json:"ry"`     // Rotation center Y
	Index  int     `json:"index"`  // Sequential index for mapping to keymap
}

// Layout represents a physical keyboard layout
type Layout struct {
	Name string        `json:"name"`
	Keys []PhysicalKey `json:"keys"`
}

// ParseKLELayout parses a KLE (keyboard-layout-editor.com) JSON format
func ParseKLELayout(data []byte, name string) (*Layout, error) {
	var raw []interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	layout := &Layout{
		Name: name,
		Keys: []PhysicalKey{},
	}

	// Current state
	currentX := 0.0
	currentY := 0.0
	currentW := 1.0
	currentH := 1.0
	currentR := 0.0
	currentRX := 0.0
	currentRY := 0.0
	keyIndex := 0

	for _, row := range raw {
		rowArray, ok := row.([]interface{})
		if !ok {
			// Skip metadata object at the start if present
			continue
		}

		// Reset X at start of each row, increment Y
		currentX = currentRX // Reset to rotation origin X
		currentY += 1.0

		for _, item := range rowArray {
			switch v := item.(type) {
			case map[string]interface{}:
				// Modifier object - update current state
				if x, ok := v["x"].(float64); ok {
					currentX += x
				}
				if y, ok := v["y"].(float64); ok {
					currentY += y
				}
				if w, ok := v["w"].(float64); ok {
					currentW = w
				}
				if h, ok := v["h"].(float64); ok {
					currentH = h
				}
				if r, ok := v["r"].(float64); ok {
					currentR = r
				}
				if rx, ok := v["rx"].(float64); ok {
					currentRX = rx
					currentX = rx // Reset X to new rotation origin
				}
				if ry, ok := v["ry"].(float64); ok {
					currentRY = ry
					currentY = ry // Reset Y to new rotation origin
				}

			case string:
				// This is a key
				key := PhysicalKey{
					X:     currentX,
					Y:     currentY - 1, // Adjust since we increment Y at row start
					W:     currentW,
					H:     currentH,
					R:     currentR,
					RX:    currentRX,
					RY:    currentRY,
					Index: keyIndex,
				}
				layout.Keys = append(layout.Keys, key)
				keyIndex++

				// Move X position for next key
				currentX += currentW

				// Reset width/height to defaults after each key
				currentW = 1.0
				currentH = 1.0
			}
		}
	}

	return layout, nil
}

// KeymapWithLayout combines a parsed keymap with a physical layout
type KeymapWithLayout struct {
	Keymap *Keymap `json:"keymap"`
	Layout *Layout `json:"layout"`
}
