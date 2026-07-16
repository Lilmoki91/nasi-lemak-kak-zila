package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"
)

// ==============================================
// 📋 RATE LIMITER
// ==============================================

type RateLimiter struct {
    mu       sync.Mutex
    requests map[string][]time.Time
    limit    int
    window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (r *RateLimiter) Allow(key string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    var valid []time.Time
    for _, t := range r.requests[key] {
        if now.Sub(t) < r.window {
            valid = append(valid, t)
        }
    }
    r.requests[key] = valid

    if len(r.requests[key]) >= r.limit {
        return false
    }

    r.requests[key] = append(r.requests[key], now)
    return true
}

// ==============================================
// 📋 CACHE
// ==============================================

type SettingsCache struct {
    mu       sync.RWMutex
    ticker   string
    memo     string
    mode     string
    lastSync time.Time
    ttl      time.Duration
}

var settingsCache = &SettingsCache{
    ttl: 30 * time.Second,
}

func LoadTickerCached() string {
    settingsCache.mu.RLock()
    if time.Since(settingsCache.lastSync) < settingsCache.ttl && settingsCache.ticker != "" {
        ticker := settingsCache.ticker
        settingsCache.mu.RUnlock()
        return ticker
    }
    settingsCache.mu.RUnlock()

    settingsCache.mu.Lock()
    defer settingsCache.mu.Unlock()

    settings, err := LoadShopSettings()
    if err != nil {
        return settingsCache.ticker
    }

    settingsCache.ticker = settings.Ticker
    settingsCache.memo = settings.Memo
    settingsCache.mode = settings.Mode
    settingsCache.lastSync = time.Now()

    return settingsCache.ticker
}

// ==============================================
// 📋 SEMAPHORE - MAX CONCURRENT AI CALLS
// ==============================================

var aiSemaphore = make(chan struct{}, 20) // Max 20 concurrent AI calls

var rateLimiter = NewRateLimiter(15, time.Minute) // 15 requests per minute

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

type ShopSettings struct {
    Mode        string `json:"mode"`
    Memo        string `json:"memo"`
    Ticker      string `json:"ticker"`
    LastUpdated string `json:"last_updated"`
}

// ==============================================
// 🔧 CORS MIDDLEWARE
// ==============================================

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        w.Header().Set("Access-Control-Max-Age", "86400")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next(w, r)
    }
}

// ==============================================
// 🔥 FIREBASE DATA LOADING
// ==============================================

func LoadShopSettings() (*ShopSettings, error) {
    siti := GetSitiAI()
    if siti.firestoreClient == nil {
        return nil, fmt.Errorf("Firestore not initialized")
    }

    ctx := context.Background()
    doc, err := siti.firestoreClient.Collection("settings").Doc("shop_settings").Get(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to load shop settings: %w", err)
    }

    data := doc.Data()
    settings := &ShopSettings{
        Mode:        getStringFromMap(data, "mode"),
        Memo:        getStringFromMap(data, "memo"),
        Ticker:      getStringFromMap(data, "ticker"),
        LastUpdated: getStringFromMap(data, "last_updated"),
    }

    if settings.Mode == "" {
        settings.Mode = "AUTO"
    }

    return settings, nil
}

// ==============================================
// 💬 CHAT HANDLER - DENGAN OPTIMASI
// ==============================================

func handleChat(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    if r.Method != "POST" {
        w.WriteHeader(http.StatusMethodNotAllowed)
        json.NewEncoder(w).Encode(ErrorResponse{
            Error:   "method_not_allowed",
            Message: "Only POST method is allowed",
        })
        return
    }

    // ⏱️ TIMEOUT CONTEXT
    ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
    defer cancel()

    // 🔒 RATE LIMIT - by IP
    clientIP := r.RemoteAddr
    if !rateLimiter.Allow(clientIP) {
        w.WriteHeader(http.StatusTooManyRequests)
        json.NewEncoder(w).Encode(ErrorResponse{
            Error:   "rate_limited",
            Message: "Too many requests. Please wait a moment.",
        })
        return
    }

    // 🔒 SEMAPHORE - limit concurrent AI calls
    select {
    case aiSemaphore <- struct{}{}:
        defer func() { <-aiSemaphore }()
    case <-time.After(5 * time.Second):
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(ErrorResponse{
            Error:   "server_busy",
            Message: "Server is busy. Please try again later.",
        })
        return
    case <-ctx.Done():
        w.WriteHeader(http.StatusRequestTimeout)
        json.NewEncoder(w).Encode(ErrorResponse{
            Error:   "timeout",
            Message: "Request timeout.",
        })
        return
    }

    // Parse request
    var req ChatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(ErrorResponse{
            Error:   "invalid_json",
            Message: "Invalid JSON format",
        })
        return
    }

    if req.Message == "" {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(ErrorResponse{
            Error:   "empty_message",
            Message: "Message cannot be empty",
        })
        return
    }

    siti := GetSitiAI()

    sessionID := req.SessionID
    if sessionID == "" {
        sessionID = "default"
    }

    // 🔥 LOG TICKER
    ticker := LoadTickerCached()
    if ticker != "" {
        log.Printf("[TICKER] Current ticker: %s", ticker)
    }

    log.Printf("[CHAT] Session: %s, Message: %s", sessionID, truncateString(req.Message, 50))

    // Check anti-spam
    if isSpam, reason := siti.CheckSpam(sessionID, req.Message); isSpam {
        log.Printf("[SPAM] Blocked: %s", reason)
        json.NewEncoder(w).Encode(ChatResponse{
            Response:  reason,
            SessionID: sessionID,
        })
        return
    }

    // Generate response with context
    responseChan := make(chan string, 1)
    errChan := make(chan error, 1)

    go func() {
        response, err := siti.Chat(req.Message, sessionID)
        if err != nil {
            errChan <- err
            return
        }
        responseChan <- response
    }()

    select {
    case response := <-responseChan:
        json.NewEncoder(w).Encode(ChatResponse{
            Response:  response,
            SessionID: sessionID,
        })
        log.Printf("[CHAT] Response sent: %d chars", len(response))
    case err := <-errChan:
        log.Printf("[ERROR] Chat failed: %v", err)
        json.NewEncoder(w).Encode(ChatResponse{
            Response:  "😔 Maaf, Siti AI ada masalah teknikal. Sila cuba lagi nanti.",
            SessionID: sessionID,
        })
    case <-ctx.Done():
        json.NewEncoder(w).Encode(ChatResponse{
            Response:  "⏳ Maaf, masa tamat. Sila cuba lagi.",
            SessionID: sessionID,
        })
    }
}

// ==============================================
// 🏠 HOME HANDLER
// ==============================================

func handleHome(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    ticker := LoadTickerCached()

    response := map[string]interface{}{
        "name":     "Zila Food AI - Siti Chatbot (GoLang)",
        "status":   "running",
        "version":  "3.1",
        "features": []string{"chat", "anti-spam", "firebase", "fallback-model", "rate-limited"},
        "runtime":  "GoLang + REST API",
        "models": map[string]string{
            "primary":  "gemini-flash-lite-latest",
            "fallback": "gemma-4-26b-a4b-it",
        },
        "concurrency": map[string]int{
            "max_concurrent_requests": 20,
            "rate_limit_per_minute":   15,
        },
    }

    if ticker != "" {
        response["ticker"] = ticker
    }

    json.NewEncoder(w).Encode(response)
}

// ==============================================
// 🏥 HEALTH CHECK
// ==============================================

func handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    siti := GetSitiAI()
    if siti.firestoreClient == nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status": "unhealthy",
            "error":  "Firestore not initialized",
        })
        return
    }

    ticker := LoadTickerCached()

    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":         "healthy",
        "timestamp":      time.Now().Unix(),
        "uptime":         time.Since(startTime).String(),
        "ticker":         ticker,
        "active_workers": 20 - len(aiSemaphore),
        "memory_usage":   getMemoryUsage(),
    })
}

var startTime time.Time

// ==============================================
// 🚀 MAIN FUNCTION
// ==============================================

func main() {
    startTime = time.Now()

    log.Println("🔧 Initializing Siti AI...")
    siti := GetSitiAI()
    if err := siti.Init(); err != nil {
        log.Fatalf("❌ Failed to initialize Siti AI: %v", err)
    }
    log.Println("✅ Siti AI initialized successfully")

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    http.HandleFunc("/api/chat", corsMiddleware(handleChat))
    http.HandleFunc("/health", handleHealth)
    http.HandleFunc("/", handleHome)

    server := &http.Server{
        Addr:         ":" + port,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 40 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    go func() {
        log.Printf("🤖 Starting AI Chatbot (Go) on port %s", port)
        log.Printf("⚡ Max concurrent AI calls: 20")
        log.Printf("⚡ Rate limit: 15 requests per minute per IP")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("❌ Server error: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("🛑 Shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("❌ Server forced to shutdown: %v", err)
    }

    log.Println("✅ Server exited gracefully")
}

// ==============================================
// 🛠️ HELPER FUNCTIONS
// ==============================================

func truncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "..."
}

func getStringFromMap(m map[string]interface{}, key string) string {
    if v, ok := m[key]; ok {
        if s, ok := v.(string); ok {
            return s
        }
    }
    return ""
}

func getMemoryUsage() string {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    return fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024)
}
