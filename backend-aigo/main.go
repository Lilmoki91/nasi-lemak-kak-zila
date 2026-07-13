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

type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

type ChatResponse struct {
	Response  string `json:"response"`
	SessionID string `json:"session_id"`
}

type MessageRecord struct {
	Timestamp time.Time
	Message   string
}

// ==============================================
// 🤖 SITI AI STRUCT
// ==============================================

type SitiAI struct {
	persona           Persona
	prompt            Prompt
	firestoreClient   *firestore.Client
	historyMutex      sync.RWMutex
	userMessageHistory map[string][]MessageRecord
	maxHistoryPerUser int
	primaryModel      string
	fallbackModel     string
}

var (
	sitiAIInstance *SitiAI
	sitiOnce       sync.Once
)

func GetSitiAI() *SitiAI {
	sitiOnce.Do(func() {
		sitiAIInstance = &SitiAI{
			userMessageHistory: make(map[string][]MessageRecord),
			maxHistoryPerUser:  20,
			primaryModel:       "gemini-flash-lite-latest",
			fallbackModel:      "gemma-4-26b-a4b-it",
		}
	})
	return sitiAIInstance
}

// ==============================================
// 🔥 INITIALIZATION
// ==============================================

func (s *SitiAI) Init() error {
	personaFile, err := os.ReadFile("persona.json")
	if err != nil {
		return fmt.Errorf("failed to read persona.json: %w", err)
	}
	if err := json.Unmarshal(personaFile, &s.persona); err != nil {
		return fmt.Errorf("failed to parse persona.json: %w", err)
	}

	promptFile, err := os.ReadFile("prompt.json")
	if err != nil {
		return fmt.Errorf("failed to read prompt.json: %w", err)
	}
	if err := json.Unmarshal(promptFile, &s.prompt); err != nil {
		return fmt.Errorf("failed to parse prompt.json: %w", err)
	}

	firebaseCreds := os.Getenv("FIREBASE_CREDENTIALS")
	if firebaseCreds != "" {
		opt := option.WithCredentialsJSON([]byte(firebaseCreds))
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err == nil {
			client, err := app.Firestore(context.Background())
			if err == nil {
				s.firestoreClient = client
				log.Println("✅ Firebase siap!")
			}
		}
	}

	log.Printf("✅ SITI AI siap! Primary: %s | Fallback: %s", s.primaryModel, s.fallbackModel)
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
		"waktu_penuh":  fmt.Sprintf("%s, %d %s %d, %s", hariMap[now.Weekday().String()], now.Day(), bulanList[now.Month()], now.Year(), now.Format("03:04 PM")),
	}
}

// ==============================================
// 🔒 ANTI-SPAM
// ==============================================

func (s *SitiAI) CheckSpam(userID, message string) (bool, string) {
	s.historyMutex.Lock()
	defer s.historyMutex.Unlock()

	currentTime := time.Now()
	if _, exists := s.userMessageHistory[userID]; !exists {
		s.userMessageHistory[userID] = []MessageRecord{}
	}

	s.userMessageHistory[userID] = append(s.userMessageHistory[userID], MessageRecord{
		Timestamp: currentTime, Message: message,
	})

	if len(s.userMessageHistory[userID]) > s.maxHistoryPerUser {
		s.userMessageHistory[userID] = s.userMessageHistory[userID][len(s.userMessageHistory[userID])-s.maxHistoryPerUser:]
	}

	history := s.userMessageHistory[userID]
	recentCount := 0
	startIdx := len(history) - 5
	if startIdx < 0 { startIdx = 0 }
	for i := startIdx; i < len(history); i++ {
		if history[i].Message == message { recentCount++ }
	}
	if recentCount >= 2 {
		return true, "🚫 *Jangan spam mesej yang sama!* 😅"
	}

	count10s := 0
	for _, msg := range history {
		if currentTime.Sub(msg.Timestamp).Seconds() < 10 { count10s++ }
	}
	if count10s >= 5 {
		return true, "⏳ *Slow slow boss! Tunggu sekejap...* 😄"
	}

	return false, ""
}

// ==============================================
// 🧠 MEMORY (FIRESTORE)
// ==============================================

func (s *SitiAI) SaveMessage(sessionID string, msg ChatMessage) error {
	if s.firestoreClient == nil { return nil }
	ctx := context.Background()
	docRef := s.firestoreClient.Collection("sessions").Doc(sessionID)
	doc, err := docRef.Get(ctx)
	if err != nil {
		session := SessionHistory{Messages: []ChatMessage{msg}, UpdatedAt: time.Now()}
		_, err = docRef.Set(ctx, session)
		return err
	}
	var session SessionHistory
	doc.DataTo(&session)
	session.Messages = append(session.Messages, msg)
	if len(session.Messages) > 10 { session.Messages = session.Messages[len(session.Messages)-10:] }
	session.UpdatedAt = time.Now()
	_, err = docRef.Set(ctx, session)
	return err
}

func (s *SitiAI) GetHistory(sessionID string) []ChatMessage {
	if s.firestoreClient == nil { return []ChatMessage{} }
	ctx := context.Background()
	doc, err := s.firestoreClient.Collection("sessions").Doc(sessionID).Get(ctx)
	if err != nil { return []ChatMessage{} }
	var session SessionHistory
	doc.DataTo(&session)
	if len(session.Messages) > 10 { return session.Messages[len(session.Messages)-10:] }
	return session.Messages
}

// ==============================================
// 📋 FIREBASE DATA
// ==============================================

func (s *SitiAI) LoadOwnerSettings() map[string]interface{} {
	if s.firestoreClient == nil { return map[string]interface{}{} }
	ctx := context.Background()
	doc, err := s.firestoreClient.Collection("settings").Doc("shop_settings").Get(ctx)
	if err != nil { return map[string]interface{}{} }
	return doc.Data()
}

func (s *SitiAI) LoadOperatingHours() map[string]interface{} {
	if s.firestoreClient == nil { return map[string]interface{}{} }
	ctx := context.Background()
	doc, err := s.firestoreClient.Collection("settings").Doc("operating_hours").Get(ctx)
	if err != nil { return map[string]interface{}{} }
	return doc.Data()
}

func (s *SitiAI) LoadMenu() []map[string]interface{} {
	if s.firestoreClient == nil { return []map[string]interface{}{} }
	ctx := context.Background()
	iter := s.firestoreClient.Collection("menu").Where("aktif", "==", true).Documents(ctx)
	defer iter.Stop()
	var menu []map[string]interface{}
	for {
		doc, err := iter.Next()
		if err != nil { break }
		data := doc.Data()
		menu = append(menu, map[string]interface{}{
			"nama": getString(data, "nama"), "desc": getString(data, "desc"),
			"harga": getFloat(data, "harga"), "featured": getBool(data, "featured"),
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
		sebab := memo; if sebab == "" { sebab = "Dibuka khas!" }
		return KedaiStatus{Status: "BUKA", Icon: "🟢", Sebab: sebab, MemoOwner: memo}
	}
	if mode == "TUTUP" {
		sebab := memo; if sebab == "" { sebab = "Ditutup." }
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
		baki := tutupMinutes - currentMinutes; jam, minit := baki/60, baki%60
		return KedaiStatus{Status: "BUKA", Icon: "🟢", Sebab: fmt.Sprintf("Beroperasi. Tutup %dj %dm lagi.", jam, minit), MemoOwner: memo}
	}
	if currentMinutes < bukaMinutes {
		baki := bukaMinutes - currentMinutes; jam, minit := baki/60, baki%60
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
		nama, _ := m["nama"].(string); desc, _ := m["desc"].(string); harga, _ := m["harga"].(float64)
		menuList.WriteString(fmt.Sprintf("- **%s** — *%s* — `RM%.2f`\n", nama, desc, harga))
	}

	hariTutupList := getIntArray(hours, "hari_tutup")
	hariNames := []string{"Ahad", "Isnin", "Selasa", "Rabu", "Khamis", "Jumaat", "Sabtu"}
	var hariTutupNames []string
	for _, d := range hariTutupList { if d >= 0 && d <= 6 { hariTutupNames = append(hariTutupNames, hariNames[d]) } }
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
📝 Memo: %s
📍 %s — %s
🗺️ Maps: %s | Waze: %s | 📲 WhatsApp: %s
📅 Hari Tutup: %s
🍗 MENU:\n%s
🎤 Gaya: %s | Catchphrase: %s
📋 Markdown: Bold **menu** | Italic *sedap* | Code 'RM5' | Bullet -
✅ %s🚫 %s`,
		s.persona.Watak.Nama, s.persona.Watak.Peranan,
		s.persona.Watak.Jantina, s.persona.Watak.Umur, strings.Join(s.persona.Watak.Gaya, ", "),
		masa["waktu_penuh"], status.Status, status.Sebab, memoOwner,
		s.persona.Kedai.Nama, s.persona.Kedai.Lokasi,
		s.persona.Kedai.GoogleMaps, s.persona.Kedai.Waze, s.persona.Kedai.Whatsapp,
		hariTutupStr, menuList.String(),
		s.persona.Watak.Sapaan, strings.Join(s.persona.Watak.Catchphrase, ", "),
		wajibList.String(), laranganList.String(),
	)
}

// ==============================================
// 🤖 CALL GEMINI API (DENGAN FALLBACK)
// ==============================================

func (s *SitiAI) callGeminiAPI(prompt string, model string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" { return "", fmt.Errorf("GEMINI_API_KEY not set") }
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	requestPayload := map[string]interface{}{
		"contents": []map[string]interface{}{{"role": "user", "parts": []map[string]string{{"text": prompt}}}},
		"generationConfig": map[string]interface{}{"temperature": 0.7, "topP": 0.45},
	}
	jsonData, _ := json.Marshal(requestPayload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(bodyBytes, &result)
	return extractTextFromResponse(result), nil
}

func (s *SitiAI) callWithFallback(fullPrompt string) (string, error) {
	log.Printf("[SitiAI] Trying primary: %s", s.primaryModel)
	response, err := s.callGeminiAPI(fullPrompt, s.primaryModel)
	if err == nil { return response, nil }
	log.Printf("[SitiAI] ⚠️ Primary failed: %v — Falling back to: %s", err, s.fallbackModel)
	return s.callGeminiAPI(fullPrompt, s.fallbackModel)
}

// ==============================================
// 💬 CHAT
// ==============================================

func (s *SitiAI) Chat(userMessage, sessionID string) (string, error) {
	if len(userMessage) > 500 { return "⚠️ Mesej terlalu panjang! Maksimum 500 aksara.", nil }
	log.Printf("[SitiAI] Processing: %s", truncate(userMessage, 50))

	history := s.GetHistory(sessionID)
	var sb strings.Builder
	sb.WriteString(s.GetSystemPrompt())
	sb.WriteString("\n\n--- HISTORY ---\n")
	for _, msg := range history { sb.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Text)) }
	sb.WriteString("\n--- CURRENT ---\nUser: " + userMessage)

	s.SaveMessage(sessionID, ChatMessage{Role: "user", Text: userMessage, Timestamp: time.Now()})
	response, err := s.callWithFallback(sb.String())
	if err != nil { return "", err }
	s.SaveMessage(sessionID, ChatMessage{Role: "model", Text: response, Timestamp: time.Now()})
	return response, nil
}

// ==============================================
// 🛠️ HELPERS
// ==============================================

func extractTextFromResponse(result map[string]interface{}) string {
	candidates, _ := result["candidates"].([]interface{})
	if len(candidates) == 0 { return "Maaf, tiada respons." }
	candidate, _ := candidates[0].(map[string]interface{})
	content, _ := candidate["content"].(map[string]interface{})
	parts, _ := content["parts"].([]interface{})
	if len(parts) == 0 { return "Maaf, tiada bahagian respons." }
	part, _ := parts[0].(map[string]interface{})
	text, _ := part["text"].(string)
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
	if v, ok := m[key]; ok { switch val := v.(type) { case float64: return val; case int: return float64(val); case int64: return float64(val) } }
	return 0
}
func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok { if b, ok := v.(bool); ok { return b } }
	return false
}
func getIntArray(m map[string]interface{}, key string) []int {
	if v, ok := m[key]; ok { if arr, ok := v.([]interface{}); ok { var result []int; for _, item := range arr { switch val := item.(type) { case float64: result = append(result, int(val)); case int: result = append(result, val); case int64: result = append(result, int(val)) } }; return result } }
	return []int{}
}
func truncate(s string, maxLen int) string { if len(s) <= maxLen { return s }; return s[:maxLen] + "..." }

// ==============================================
// 🌐 HTTP SERVER
// ==============================================

func handleChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" { w.WriteHeader(http.StatusOK); return }
	if r.Method != "POST" { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }

	var req ChatRequest
	json.NewDecoder(r.Body).Decode(&req)
	if req.SessionID == "" { req.SessionID = fmt.Sprintf("session-%d", time.Now().UnixNano()) }

	siti := GetSitiAI()
	if isSpam, reason := siti.CheckSpam(req.SessionID, req.Message); isSpam {
		json.NewEncoder(w).Encode(ChatResponse{Response: reason, SessionID: req.SessionID})
		return
	}

	response, err := siti.Chat(req.Message, req.SessionID)
	if err != nil { response = "😔 Maaf, SITI AI mengalami gangguan." }
	json.NewEncoder(w).Encode(ChatResponse{Response: response, SessionID: req.SessionID})
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name": "SITI AI", "version": "3.0.0",
		"primary": "gemini-flash-lite-latest", "fallback": "gemma-4-26b-a4b-it", "status": "running",
	})
}

func main() {
	siti := GetSitiAI()
	if err := siti.Init(); err != nil { log.Fatalf("Init failed: %v", err) }
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	http.HandleFunc("/api/chat", handleChat)
	http.HandleFunc("/", handleHome)
	log.Printf("🤖 SITI AI running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
