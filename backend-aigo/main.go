package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

// ==============================================
// 📋 DATA STRUCTURES
// ==============================================

type ChatRequest struct {
    Message   string `json:"message"`
    SessionID string `json:"session_id"`
}

type ChatResponse struct {
    Response  string `json:"response"`
    SessionID string `json:"session_id"`
}

type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message"`
}

// ==============================================
// 🔧 CORS MIDDLEWARE
// ==============================================

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Set CORS headers
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

        // Handle preflight request
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        // Call next handler
        next(w, r)
    }
}

// ==============================================
// 💬 CHAT HANDLER
// ==============================================

func handleChat(w http.ResponseWriter, r *http.Request) {
    // 🔒 Set headers FIRST
    w.Header().Set("Content-Type", "application/json")

    // Only allow POST
    if r.Method != "POST" {
        w.WriteHeader(http.StatusMethodNotAllowed)
        json.NewEncoder(w).Encode(ErrorResponse{
            Error:   "method_not_allowed",
            Message: "Only POST method is allowed",
        })
        return
    }

    // Parse request body
    var req ChatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(ErrorResponse{
            Error:   "invalid_json",
            Message: "Invalid JSON format",
        })
        return
    }

    // Validate message
    if req.Message == "" {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(ErrorResponse{
            Error:   "empty_message",
            Message: "Message cannot be empty",
        })
        return
    }

    // Get Siti AI instance
    siti := GetSitiAI()

    // Set default session ID
    sessionID := req.SessionID
    if sessionID == "" {
        sessionID = "default"
    }

    log.Printf("[CHAT] Session: %s, Message: %s", sessionID, truncate(req.Message, 50))

    // Check anti-spam
    if isSpam, reason := siti.CheckSpam(sessionID, req.Message); isSpam {
        log.Printf("[SPAM] Blocked: %s", reason)
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
        response = "😔 Maaf, Siti AI ada masalah teknikal. Sila cuba lagi nanti."
    }

    // Send response
    json.NewEncoder(w).Encode(ChatResponse{
        Response:  response,
        SessionID: sessionID,
    })

    log.Printf("[CHAT] Response sent: %d chars", len(response))
}

// ==============================================
// 🏠 HOME HANDLER
// ==============================================

func handleHome(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "name":     "Zila Food AI - Siti Chatbot (GoLang)",
        "status":   "running",
        "version":  "3.1",
        "features": []string{"chat", "anti-spam", "firebase", "fallback-model"},
        "runtime":  "GoLang + REST API",
        "models": map[string]string{
            "primary":  "gemini-flash-lite-latest",
            "fallback": "gemma-4-26b-a4b-it",
        },
    })
}

// ==============================================
// 🏥 HEALTH CHECK (UNTUK RENDER)
// ==============================================

func handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    
    // Check if Siti AI is initialized
    siti := GetSitiAI()
    if siti.firestoreClient == nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status": "unhealthy",
            "error":  "Firestore not initialized",
        })
        return
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now().Unix(),
        "uptime":    time.Since(startTime).String(),
    })
}

var startTime time.Time

// ==============================================
// 🚀 MAIN FUNCTION
// ==============================================

func main() {
    startTime = time.Now()

    // Initialize Siti AI
    log.Println("🔧 Initializing Siti AI...")
    siti := GetSitiAI()
    if err := siti.Init(); err != nil {
        log.Fatalf("❌ Failed to initialize Siti AI: %v", err)
    }
    log.Println("✅ Siti AI initialized successfully")

    // Get port from environment
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    // Setup routes
    http.HandleFunc("/api/chat", corsMiddleware(handleChat))
    http.HandleFunc("/health", handleHealth)
    http.HandleFunc("/", handleHome)

    // Create server with timeouts
    server := &http.Server{
        Addr:         ":" + port,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    // 🔥 GRACEFUL SHUTDOWN
    go func() {
        log.Printf("🤖 Starting AI Chatbot (Go) on port %s", port)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("❌ Server error: %v", err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("🛑 Shutting down server...")

    // Give outstanding requests 30 seconds to complete
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("❌ Server forced to shutdown: %v", err)
    }

    log.Println("✅ Server exited gracefully")
}

// ==============================================
// 🛠️ HELPER FUNCTION
// ==============================================

func truncate(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "..."
}
