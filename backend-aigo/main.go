package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

type ChatResponse struct {
	Response  string `json:"response"`
	SessionID string `json:"session_id"`
}

// CORS middleware
func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Get or create session
	siti := GetSitiAI()
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "default"
	}

	// Check anti-spam
	if isSpam, reason := siti.CheckSpam(sessionID, req.Message); isSpam {
		json.NewEncoder(w).Encode(ChatResponse{
			Response:  reason,
			SessionID: sessionID,
		})
		return
	}

	// Generate response
	response, err := siti.Chat(req.Message, sessionID)
	if err != nil {
		log.Printf("[ERROR] Chat failed: %v", err)
		response = "😔 Maaf, Siti AI ada masalah teknikal."
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Response:  response,
		SessionID: sessionID,
	})
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":     "Zila Food AI - Siti Chatbot (GoLang)",
		"status":   "running",
		"version":  "3.0",
		"features": []string{"chat", "anti-spam", "firebase"},
		"runtime":  "GoLang + REST API",
	})
}

func main() {
	// Initialize Siti AI
	siti := GetSitiAI()
	if err := siti.Init(); err != nil {
		log.Fatalf("Failed to initialize Siti AI: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/api/chat", handleChat)
	http.HandleFunc("/", handleHome)

	log.Printf("🤖 Starting AI Chatbot (Go) on port %s", port)
	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
