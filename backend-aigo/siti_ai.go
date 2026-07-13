package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strings"
    "sync"
    "time"

    "cloud.google.com/go/firestore"
    firebase "firebase.google.com/go/v4"
    "google.golang.org/api/option"
)

// ==============================================
// 📋 DATA STRUCTURES
// ==============================================

type Persona struct {
    Watak struct {
        Nama        string   `json:"nama"`
        Peranan     string   `json:"peranan"`
        Jantina     string   `json:"jantina"`
        Umur        int      `json:"umur"`
        Gaya        []string `json:"gaya"`
        Sapaan      string   `json:"sapaan"`
        Catchphrase []string `json:"catchphrase"`
    } `json:"watak"`
    Kedai struct {
        Nama       string `json:"nama"`
        Lokasi     string `json:"lokasi"`
        GoogleMaps string `json:"google_maps"`
        Waze       string `json:"waze"`
        Whatsapp   string `json:"whatsapp"`
    } `json:"kedai"`
}

type Prompt struct {
    Wajib    []string `json:"wajib"`
    Larangan []string `json:"larangan"`
}

type KedaiStatus struct {
    Status    string `json:"status"`
    Icon      string `json:"icon"`
    Sebab     string `json:"sebab"`
    MemoOwner string `json:"memo_owner"`
}

type ChatMessage struct {
    Role      string    `json:"role"`
    Text      string    `json:"text"`
    Timestamp time.Time `json:"timestamp"`
}

type SessionHistory struct {
    Messages  []ChatMessage `json:"messages"`
    UpdatedAt time.Time     `json:"updated_at"`
}

// ==============================================
// 🤖 SITI AI STRUCT
// ==============================================

type SitiAI struct {
    persona         Persona
    prompt          Prompt
    firestoreClient *firestore.Client
    historyMutex    sync.RWMutex
    // 🔥 MODEL PRIORITY
    primaryModel   string // Gemini Flash Lite (default)
    fallbackModel  string // Gemma 4 (fallback)
}

var (
    sitiAIInstance *SitiAI
    sitiOnce       sync.Once
)

func GetSitiAI() *SitiAI {
    sitiOnce.Do(func() {
        sitiAIInstance = &SitiAI{
            primaryModel:  "gemini-flash-lite-latest", // 🔥 GEMINI FLASH LITE
            fallbackModel: "gemma-4-26b-a4b-it",       // 🔥 FALLBACK GEMMA 4
        }
    })
    return sitiAIInstance
}

// ==============================================
// 🔥 INITIALIZATION
// ==============================================

func (s *SitiAI) Init() error {
    // Load persona.json
    personaFile, err := os.ReadFile("persona.json")
    if err != nil {
        return fmt.Errorf("failed to read persona.json: %w", err)
    }
    if err := json.Unmarshal(personaFile, &s.persona); err != nil {
        return fmt.Errorf("failed to parse persona.json: %w", err)
    }

    // Load prompt.json
    promptFile, err := os.ReadFile("prompt.json")
    if err != nil {
        return fmt.Errorf("failed to read prompt.json: %w", err)
    }
    if err := json.Unmarshal(promptFile, &s.prompt); err != nil {
        return fmt.Errorf("failed to parse prompt.json: %w", err)
    }

    // Initialize Firebase
    firebaseCreds := os.Getenv("FIREBASE_CREDENTIALS")
    if firebaseCreds == "" {
        log.Println("⚠️ FIREBASE_CREDENTIALS not set — running without Firebase")
        return nil
    }

    opt := option.WithCredentialsJSON([]byte(firebaseCreds))
    app, err := firebase.NewApp(context.Background(), nil, opt)
    if err != nil {
        return fmt.Errorf("failed to initialize Firebase: %w", err)
    }

    client, err := app.Firestore(context.Background())
    if err != nil {
        return fmt.Errorf("failed to get Firestore client: %w", err)
    }
    s.firestoreClient = client

    log.Println("✅ SITI AI initialized — Primary:", s.primaryModel, "| Fallback:", s.fallbackModel)
    return nil
}

// ==============================================
// ⏰ MALAYSIA TIME
// ==============================================

func (s *SitiAI) GetMalaysiaTime() map[string]string {
    loc, _ := time.LoadLocation("Asia/Kuala_Lumpur")
    now := time.Now().In(loc)

    hariMap := map[string]string{
        "Monday": "Isnin", "Tuesday": "Selasa", "Wednesday": "Rabu",
        "Thursday": "Khamis", "Friday": "Jumaat", "Saturday": "Sabtu", "Sunday": "Ahad",
    }
    bulanList := []string{"", "Januari", "Februari", "Mac", "April", "Mei", "Jun",
        "Julai", "Ogos", "September", "Oktober", "November", "Disember"}

    return map[string]string{
        "jam_12h":      now.Format("03:04 PM"),
        "jam_24h":      now.Format("15:04"),
        "hari":         hariMap[now.Weekday().String()],
        "hari_num":     fmt.Sprintf("%d", now.Weekday()),
        "tarikh_penuh": fmt.Sprintf("%d %s %d", now.Day(), bulanList[now.Month()], now.Year()),
        "waktu_penuh": fmt.Sprintf("%s, %d %s %d, %s",
            hariMap[now.Weekday().String()], now.Day(), bulanList[now.Month()], now.Year(), now.Format("03:04 PM")),
    }
}

// ==============================================
// 🧠 MEMORY MANAGEMENT (10 MESEJ TERAKHIR)
// ==============================================

func (s *SitiAI) SaveMessage(sessionID string, msg ChatMessage) error {
    if s.firestoreClient == nil {
        return nil
    }

    ctx := context.Background()
    docRef := s.firestoreClient.Collection("sessions").Doc(sessionID)

    doc, err := docRef.Get(ctx)
    if err != nil {
        session := SessionHistory{
            Messages:  []ChatMessage{msg},
            UpdatedAt: time.Now(),
        }
        _, err = docRef.Set(ctx, session)
        return err
    }

    var session SessionHistory
    if err := doc.DataTo(&session); err != nil {
        session = SessionHistory{
            Messages:  []ChatMessage{msg},
            UpdatedAt: time.Now(),
        }
        _, err = docRef.Set(ctx, session)
        return err
    }

    session.Messages = append(session.Messages, msg)
    if len(session.Messages) > 10 {
        session.Messages = session.Messages[len(session.Messages)-10:]
    }
    session.UpdatedAt = time.Now()
    _, err = docRef.Set(ctx, session)
    return err
}

func (s *SitiAI) GetHistory(sessionID string) []ChatMessage {
    if s.firestoreClient == nil {
        return []ChatMessage{}
    }

    ctx := context.Background()
    doc, err := s.firestoreClient.Collection("sessions").Doc(sessionID).Get(ctx)
    if err != nil {
        return []ChatMessage{}
    }

    var session SessionHistory
    doc.DataTo(&session)

    if len(session.Messages) > 10 {
        return session.Messages[len(session.Messages)-10:]
    }
    return session.Messages
}

// ==============================================
// 📋 FIREBASE DATA LOADING
// ==============================================

func (s *SitiAI) LoadOwnerSettings() map[string]interface{} {
    if s.firestoreClient == nil {
        return map[string]interface{}{}
    }
    ctx := context.Background()
    doc, err := s.firestoreClient.Collection("settings").Doc("shop_settings").Get(ctx)
    if err != nil {
        return map[string]interface{}{}
    }
    return doc.Data()
}

func (s *SitiAI) LoadOperatingHours() map[string]interface{} {
    if s.firestoreClient == nil {
        return map[string]interface{}{}
    }
    ctx := context.Background()
    doc, err := s.firestoreClient.Collection("settings").Doc("operating_hours").Get(ctx)
    if err != nil {
        return map[string]interface{}{}
    }
    return doc.Data()
}

func (s *SitiAI) LoadMenu() []map[string]interface{} {
    if s.firestoreClient == nil {
        return []map[string]interface{}{}
    }
    ctx := context.Background()
    iter := s.firestoreClient.Collection("menu").Where("aktif", "==", true).Documents(ctx)
    defer iter.Stop()

    var menu []map[string]interface{}
    for {
        doc, err := iter.Next()
        if err != nil {
            break
        }
        data := doc.Data()
        menu = append(menu, map[string]interface{}{
            "nama":     getString(data, "nama"),
            "desc":     getString(data, "desc"),
            "harga":    getFloat(data, "harga"),
            "featured": getBool(data, "featured"),
        })
    }
    return menu
}

// ==============================================
// 🟢 STATUS KEDAI
// ==============================================

func (s *SitiAI) CheckKedaiStatus() KedaiStatus {
    owner := s.LoadOwnerSettings()
    hours := s.LoadOperatingHours()

    mode, _ := owner["mode"].(string)
    memo, _ := owner["memo"].(string)
    waktuBuka, _ := hours["buka"].(string)
    waktuTutup, _ := hours["tutup"].(string)

    if mode == "BUKA" {
        sebab := memo
        if sebab == "" { sebab = "Dibuka khas!" }
        return KedaiStatus{Status: "BUKA", Icon: "🟢", Sebab: sebab, MemoOwner: memo}
    }
    if mode == "TUTUP" {
        sebab := memo
        if sebab == "" { sebab = "Ditutup." }
        return KedaiStatus{Status: "TUTUP", Icon: "🔴", Sebab: sebab, MemoOwner: memo}
    }

    if waktuBuka == "" || waktuTutup == "" {
        return KedaiStatus{Status: "TUTUP", Icon: "🔴", Sebab: "Waktu operasi belum ditetapkan."}
    }

    bukaMinutes := parseTimeToMinutes(waktuBuka)
    tutupMinutes := parseTimeToMinutes(waktuTutup)
    if tutupMinutes == 0 { tutupMinutes = 24 * 60 }

    loc, _ := time.LoadLocation("Asia/Kuala_Lumpur")
    now := time.Now().In(loc)
    currentMinutes := now.Hour()*60 + now.Minute()
    hariNum := int(now.Weekday())

    hariTutupList := getIntArray(hours, "hari_tutup")
    hariNames := []string{"Ahad", "Isnin", "Selasa", "Rabu", "Khamis", "Jumaat", "Sabtu"}

    for _, h := range hariTutupList {
        if h == hariNum {
            return KedaiStatus{Status: "TUTUP", Icon: "🔴", Sebab: fmt.Sprintf("Hari %s — tutup.", hariNames[hariNum]), MemoOwner: memo}
        }
    }

    if bukaMinutes <= currentMinutes && currentMinutes < tutupMinutes {
        baki := tutupMinutes - currentMinutes
        jam, minit := baki/60, baki%60
        return KedaiStatus{Status: "BUKA", Icon: "🟢", Sebab: fmt.Sprintf("Beroperasi. Tutup %dj %dm lagi.", jam, minit), MemoOwner: memo}
    }

    if currentMinutes < bukaMinutes {
        baki := bukaMinutes - currentMinutes
        jam, minit := baki/60, baki%60
        return KedaiStatus{Status: "TUTUP", Icon: "🔴", Sebab: fmt.Sprintf("Belum buka. Buka %dj %dm lagi.", jam, minit), MemoOwner: memo}
    }

    return KedaiStatus{Status: "TUTUP", Icon: "🔴", Sebab: "Sudah tutup.", MemoOwner: memo}
}

// ==============================================
// 📝 SYSTEM PROMPT
// ==============================================

func (s *SitiAI) GetSystemPrompt() string {
    masa := s.GetMalaysiaTime()
    status := s.CheckKedaiStatus()
    menuItems := s.LoadMenu()
    owner := s.LoadOwnerSettings()
    hours := s.LoadOperatingHours()

    var menuList strings.Builder
    for _, m := range menuItems {
        nama, _ := m["nama"].(string)
        desc, _ := m["desc"].(string)
        harga, _ := m["harga"].(float64)
        menuList.WriteString(fmt.Sprintf("- **%s** — *%s* — `RM%.2f`\n", nama, desc, harga))
    }

    hariTutupList := getIntArray(hours, "hari_tutup")
    hariNames := []string{"Ahad", "Isnin", "Selasa", "Rabu", "Khamis", "Jumaat", "Sabtu"}
    var hariTutupNames []string
    for _, d := range hariTutupList {
        if d >= 0 && d <= 6 {
            hariTutupNames = append(hariTutupNames, hariNames[d])
        }
    }
    hariTutupStr := "Tiada"
    if len(hariTutupNames) > 0 { hariTutupStr = strings.Join(hariTutupNames, ", ") }

    memoOwner, _ := owner["memo"].(string)
    if memoOwner == "" { memoOwner = "Tiada" }

    var wajibList, laranganList strings.Builder
    for _, w := range s.prompt.Wajib { wajibList.WriteString("- " + w + "\n") }
    for _, l := range s.prompt.Larangan { laranganList.WriteString("- " + l + "\n") }

    return fmt.Sprintf(`Anda adalah %s, %s.
Persona: %s Melayu %d tahun, %s.

⏰ Sekarang: %s
🟢 Status: **%s** — %s
📝 Memo Owner: %s

📍 %s — %s
🗺️ Maps: %s | Waze: %s
📲 WhatsApp: %s
📅 Hari Tutup: %s

🍗 MENU:
%s

🎤 Gaya: %s | Catchphrase: %s
📋 Markdown: Bold **menu** | Italic *sedap* | Code 'RM5' | Bullet -
✅ %s🚫 %s`,
        s.persona.Watak.Nama, s.persona.Watak.Peranan,
        s.persona.Watak.Jantina, s.persona.Watak.Umur, strings.Join(s.persona.Watak.Gaya, ", "),
        masa["waktu_penuh"],
        status.Status, status.Sebab,
        memoOwner,
        s.persona.Kedai.Nama, s.persona.Kedai.Lokasi,
        s.persona.Kedai.GoogleMaps, s.persona.Kedai.Waze,
        s.persona.Kedai.Whatsapp,
        hariTutupStr,
        menuList.String(),
        s.persona.Watak.Sapaan, strings.Join(s.persona.Watak.Catchphrase, ", "),
        wajibList.String(), laranganList.String(),
    )
}

// ==============================================
// 🤖 CALL GEMINI API (DENGAN FALLBACK)
// ==============================================

func (s *SitiAI) callGeminiAPI(prompt string, model string) (string, error) {
    apiKey := os.Getenv("GEMINI_API_KEY")
    if apiKey == "" {
        return "", fmt.Errorf("GEMINI_API_KEY not set")
    }

    url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

    requestPayload := map[string]interface{}{
        "contents": []map[string]interface{}{
            {
                "role": "user",
                "parts": []map[string]string{
                    {"text": prompt},
                },
            },
        },
        "generationConfig": map[string]interface{}{
            "temperature": 0.7,
            "topP":        0.45,
            "thinkingConfig": map[string]interface{}{
                "thinkingLevel": "MINIMAL",
            },
        },
    }

    jsonData, err := json.Marshal(requestPayload)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
    }
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
    }

    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    var result map[string]interface{}
    if err := json.Unmarshal(bodyBytes, &result); err != nil {
        return "", err
    }

    return extractTextFromResponse(result), nil
}

// 🔥 CALL WITH FALLBACK
func (s *SitiAI) callWithFallback(fullPrompt string) (string, error) {
    // Cuba primary model dulu (Gemini Flash Lite)
    log.Printf("[SitiAI] Trying primary model: %s", s.primaryModel)
    response, err := s.callGeminiAPI(fullPrompt, s.primaryModel)
    if err == nil {
        log.Printf("[SitiAI] ✅ Primary model success: %s", s.primaryModel)
        return response, nil
    }

    // Fallback ke Gemma 4
    log.Printf("[SitiAI] ⚠️ Primary model failed: %v", err)
    log.Printf("[SitiAI] 🔄 Falling back to: %s", s.fallbackModel)

    response, err = s.callGeminiAPI(fullPrompt, s.fallbackModel)
    if err != nil {
        return "", fmt.Errorf("both models failed — primary: %v, fallback: %v", err, err)
    }

    log.Printf("[SitiAI] ✅ Fallback model success: %s", s.fallbackModel)
    return response, nil
}

// ==============================================
// 💬 CHAT
// ==============================================

func (s *SitiAI) Chat(userMessage, sessionID string) (string, error) {
    const maxInput = 500
    if len(userMessage) > maxInput {
        return fmt.Sprintf("⚠️ Mesej terlalu panjang! Maksimum %d aksara.", maxInput), nil
    }

    log.Printf("[SitiAI] Processing: %s", truncate(userMessage, 50))

    // Get history
    history := s.GetHistory(sessionID)

    // Build full prompt
    var sb strings.Builder
    sb.WriteString(s.GetSystemPrompt())
    sb.WriteString("\n\n--- HISTORY ---\n")
    for _, msg := range history {
        sb.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Text))
    }
    sb.WriteString("\n--- CURRENT ---\n")
    sb.WriteString("User: " + userMessage)

    fullPrompt := sb.String()

    // Save user message
    s.SaveMessage(sessionID, ChatMessage{
        Role: "user", Text: userMessage, Timestamp: time.Now(),
    })

    // Call Gemini dengan fallback
    response, err := s.callWithFallback(fullPrompt)
    if err != nil {
        return "", err
    }

    // Save AI response
    s.SaveMessage(sessionID, ChatMessage{
        Role: "model", Text: response, Timestamp: time.Now(),
    })

    log.Printf("[SitiAI] Response: %d chars", len(response))
    return response, nil
}

// ==============================================
// 🛠️ HELPERS
// ==============================================

func extractTextFromResponse(result map[string]interface{}) string {
    candidates, ok := result["candidates"].([]interface{})
    if !ok || len(candidates) == 0 { return "Maaf, tiada respons." }
    candidate, ok := candidates[0].(map[string]interface{})
    if !ok { return "Maaf, format respons tidak sah." }
    content, ok := candidate["content"].(map[string]interface{})
    if !ok { return "Maaf, kandungan respons tidak sah." }
    parts, ok := content["parts"].([]interface{})
    if !ok || len(parts) == 0 { return "Maaf, tiada bahagian respons." }
    part, ok := parts[0].(map[string]interface{})
    if !ok { return "Maaf, format bahagian tidak sah." }
    text, ok := part["text"].(string)
    if !ok { return "Maaf, teks respons tidak dijumpai." }
    return text
}

func parseTimeToMinutes(timeStr string) int {
    parts := strings.Split(timeStr, ":")
    if len(parts) != 2 { return 0 }
    var h, m int
    fmt.Sscanf(parts[0], "%d", &h)
    fmt.Sscanf(parts[1], "%d", &m)
    return h*60 + m
}

func getString(m map[string]interface{}, key string) string {
    if v, ok := m[key]; ok { if s, ok := v.(string); ok { return s } }
    return ""
}
func getFloat(m map[string]interface{}, key string) float64 {
    if v, ok := m[key]; ok {
        switch val := v.(type) {
        case float64: return val
        case int: return float64(val)
        case int64: return float64(val)
        }
    }
    return 0
}
func getBool(m map[string]interface{}, key string) bool {
    if v, ok := m[key]; ok { if b, ok := v.(bool); ok { return b } }
    return false
}
func getIntArray(m map[string]interface{}, key string) []int {
    if v, ok := m[key]; ok {
        if arr, ok := v.([]interface{}); ok {
            var result []int
            for _, item := range arr {
                switch val := item.(type) {
                case float64: result = append(result, int(val))
                case int: result = append(result, val)
                case int64: result = append(result, int(val))
                }
            }
            return result
        }
    }
    return []int{}
}
func truncate(s string, maxLen int) string {
    if len(s) <= maxLen { return s }
    return s[:maxLen] + "..."
}

// ==============================================
// 🌐 HTTP SERVER
// ==============================================

type ChatRequest struct {
    Message   string `json:"message"`
    SessionID string `json:"session_id"`
}

type ChatResponse struct {
    Response  string `json:"response"`
    SessionID string `json:"session_id"`
}

func main() {
    siti := GetSitiAI()
    if err := siti.Init(); err != nil {
        log.Fatalf("Failed to initialize: %v", err)
    }

    http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

        if r.Method == "OPTIONS" { w.WriteHeader(http.StatusOK); return }
        if r.Method != "POST" { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }

        var req ChatRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest); return
        }
        if strings.TrimSpace(req.Message) == "" {
            http.Error(w, "Message required", http.StatusBadRequest); return
        }
        if req.SessionID == "" {
            req.SessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
        }

        response, err := siti.Chat(req.Message, req.SessionID)
        if err != nil {
            log.Printf("Chat error: %v", err)
            response = "😔 Maaf, SITI AI mengalami gangguan."
        }

        json.NewEncoder(w).Encode(ChatResponse{Response: response, SessionID: req.SessionID})
    })

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "name":    "SITI AI",
            "version": "3.0.0",
            "primary": "gemini-flash-lite-latest",
            "fallback": "gemma-4-26b-a4b-it",
            "status":  "running",
        })
    })

    port := os.Getenv("PORT")
    if port == "" { port = "8080" }

    log.Printf("🤖 SITI AI running on :%s (Primary: %s | Fallback: %s)", port, siti.primaryModel, siti.fallbackModel)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
