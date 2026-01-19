package main

import (
	"log"
	"net/http"

	"keyviewer/internal/api"
)

func main() {
	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// API routes - Keymaps
	http.HandleFunc("/api/keymap", api.HandleKeymap)
	http.HandleFunc("/api/keymaps", api.HandleKeymaps)
	http.HandleFunc("/api/keymap/", api.HandleKeymapByName)

	// API routes - Layouts
	http.HandleFunc("/api/layout", api.HandleLayout)
	http.HandleFunc("/api/layouts", api.HandleLayouts)
	http.HandleFunc("/api/layout/", api.HandleLayoutByName)

	log.Println("KeyViewer server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
