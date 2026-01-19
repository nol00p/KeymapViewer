package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"keyviewer/internal/parser"
)

const keymapsDir = "keymaps"
const layoutsDir = "layouts"

func init() {
	os.MkdirAll(keymapsDir, 0755)
	os.MkdirAll(layoutsDir, 0755)
}

// HandleKeymap handles POST requests to upload and parse a keymap
func HandleKeymap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("keymap")
	if err != nil {
		http.Error(w, "No keymap file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	name := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))

	keymap, err := parser.ParseKeymap(string(content), name)
	if err != nil {
		http.Error(w, "Failed to parse keymap: "+err.Error(), http.StatusBadRequest)
		return
	}

	jsonData, err := json.MarshalIndent(keymap, "", "  ")
	if err != nil {
		http.Error(w, "Failed to serialize keymap", http.StatusInternalServerError)
		return
	}

	jsonPath := filepath.Join(keymapsDir, name+".json")
	err = os.WriteFile(jsonPath, jsonData, 0644)
	if err != nil {
		http.Error(w, "Failed to save keymap", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// HandleKeymaps handles GET requests to list available keymaps
func HandleKeymaps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entries, err := os.ReadDir(keymapsDir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			names = append(names, name)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(names)
}

// HandleKeymapImport handles POST requests to import a keymap JSON directly
func HandleKeymapImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Validate JSON structure
	var keymap parser.Keymap
	if err := json.Unmarshal(body, &keymap); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if keymap.Name == "" {
		http.Error(w, "Keymap name is required", http.StatusBadRequest)
		return
	}
	if len(keymap.Layers) == 0 {
		http.Error(w, "Keymap must have at least one layer", http.StatusBadRequest)
		return
	}

	// Re-marshal with proper formatting
	jsonData, err := json.MarshalIndent(keymap, "", "  ")
	if err != nil {
		http.Error(w, "Failed to serialize keymap", http.StatusInternalServerError)
		return
	}

	// Save to file
	jsonPath := filepath.Join(keymapsDir, keymap.Name+".json")
	err = os.WriteFile(jsonPath, jsonData, 0644)
	if err != nil {
		http.Error(w, "Failed to save keymap", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// HandleKeymapByName handles GET and PATCH requests for a specific keymap
func HandleKeymapByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/keymap/")
	if name == "" {
		http.Error(w, "Keymap name required", http.StatusBadRequest)
		return
	}

	jsonPath := filepath.Join(keymapsDir, name+".json")

	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(jsonPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "Keymap not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to read keymap", http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)

	case http.MethodPatch:
		// Update custom key name
		var update struct {
			LayerIndex int    `json:"layerIndex"`
			KeyIndex   int    `json:"keyIndex"`
			CustomName string `json:"customName"`
		}

		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Read existing keymap
		data, err := os.ReadFile(jsonPath)
		if err != nil {
			http.Error(w, "Keymap not found", http.StatusNotFound)
			return
		}

		var keymap parser.Keymap
		if err := json.Unmarshal(data, &keymap); err != nil {
			http.Error(w, "Failed to parse keymap", http.StatusInternalServerError)
			return
		}

		// Validate layer index
		if update.LayerIndex < 0 || update.LayerIndex >= len(keymap.Layers) {
			http.Error(w, "Invalid layer index", http.StatusBadRequest)
			return
		}

		// Initialize CustomNames map if nil
		if keymap.Layers[update.LayerIndex].CustomNames == nil {
			keymap.Layers[update.LayerIndex].CustomNames = make(map[string]string)
		}

		// Update or remove custom name
		keyIdx := strconv.Itoa(update.KeyIndex)
		if update.CustomName == "" {
			delete(keymap.Layers[update.LayerIndex].CustomNames, keyIdx)
		} else {
			keymap.Layers[update.LayerIndex].CustomNames[keyIdx] = update.CustomName
		}

		// Save updated keymap
		jsonData, err := json.MarshalIndent(keymap, "", "  ")
		if err != nil {
			http.Error(w, "Failed to serialize keymap", http.StatusInternalServerError)
			return
		}

		if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
			http.Error(w, "Failed to save keymap", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleLayout handles POST requests to upload a KLE layout
func HandleLayout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("layout")
	if err != nil {
		http.Error(w, "No layout file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	name := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))

	layout, err := parser.ParseKLELayout(content, name)
	if err != nil {
		http.Error(w, "Failed to parse layout: "+err.Error(), http.StatusBadRequest)
		return
	}

	jsonData, err := json.MarshalIndent(layout, "", "  ")
	if err != nil {
		http.Error(w, "Failed to serialize layout", http.StatusInternalServerError)
		return
	}

	jsonPath := filepath.Join(layoutsDir, name+".json")
	err = os.WriteFile(jsonPath, jsonData, 0644)
	if err != nil {
		http.Error(w, "Failed to save layout", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// HandleLayouts handles GET requests to list available layouts
func HandleLayouts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entries, err := os.ReadDir(layoutsDir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			names = append(names, name)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(names)
}

// HandleLayoutByName handles GET requests for a specific layout
func HandleLayoutByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/layout/")
	if name == "" {
		http.Error(w, "Layout name required", http.StatusBadRequest)
		return
	}

	jsonPath := filepath.Join(layoutsDir, name+".json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Layout not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to read layout", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
