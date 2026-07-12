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

type MessageRecord struct {
	Timestamp time.Time
	Message   string
}

// ==============================================
// 🤖 SITI AI STRUCT
// ==============================================

type SitiAI struct {
	persona            Persona
	prompt             Prompt
	firestoreClient    *firestore.Client
	userMessageHistory map[string][]MessageRecord
	historyMutex       sync.RWMutex
	maxHistoryPerUser  int
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
		return fmt.Errorf("FIREBASE_CREDENTIALS environment variable not set")
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

	log.Println("✅ Siti AI initialized successfully")
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
		"jam_12h":       now.Format("03:04 PM"),
		"jam_24h":       now.Format("15:04"),
		"hari":          hariMap[now.Weekday().String()],
		"hari_num":      fmt.Sprintf("%d", now.Weekday()),
		"tarikh_penuh":  fmt.Sprintf("%d %s %d", now.Day(), bulanList[now.Month()], now.Year()),
		"waktu_penuh":   fmt.Sprintf("%s, %d %s %d, %s",
			hariMap[now.Weekday().String()], now.Day(), bulanList[now.Month()], now.Year(), now.Format("03:04 PM")),
	}
}

// ==============================================
// 📋 FIREBASE DATA LOADING
// ==============================================

func (s *SitiAI) LoadOwnerSettings() map[string]interface{} {
	ctx := context.Background()
	doc, err := s.firestoreClient.Collection("settings").Doc("shop_settings").Get(ctx)
	if err != nil {
		log.Printf("[SitiAI] Error loading owner settings: %v", err)
		return map[string]interface{}{}
	}
	return doc.Data()
}

func (s *SitiAI) LoadOperatingHours() map[string]interface{} {
	ctx := context.Background()
	doc, err := s.firestoreClient.Collection("settings").Doc("operating_hours").Get(ctx)
	if err != nil {
		log.Printf("[SitiAI] Error loading operating hours: %v", err)
		return map[string]interface{}{}
	}
	return doc.Data()
}

func (s *SitiAI) LoadMenu() []map[string]interface{} {
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

	log.Printf("[SitiAI] Mode dari Firebase: %s", mode)

	if mode == "BUKA" {
		sebab := memo
		if sebab == "" {
			sebab = "Dibuka khas!"
		}
		return KedaiStatus{Status: "BUKA", Icon: "🟢", Sebab: sebab, MemoOwner: memo}
	}
	if mode == "TUTUP" {
		sebab := memo
		if sebab == "" {
			sebab = "Ditutup."
		}
		return KedaiStatus{Status: "TUTUP", Icon: "🔴", Sebab: sebab, MemoOwner: memo}
	}

	if waktuBuka == "" || waktuTutup == "" {
		return KedaiStatus{Status: "TUTUP", Icon: "🔴", Sebab: "Waktu operasi belum ditetapkan."}
	}

	// Parse waktu
	bukaMinutes := parseTimeToMinutes(waktuBuka)
	tutupMinutes := parseTimeToMinutes(waktuTutup)
	if tutupMinutes == 0 {
		tutupMinutes = 24 * 60
	}

	loc, _ := time.LoadLocation("Asia/Kuala_Lumpur")
	now := time.Now().In(loc)
	currentMinutes := now.Hour()*60 + now.Minute()
	hariNum := int(now.Weekday())

	hariTutupList := getIntArray(hours, "hari_tutup")
	hariNames := []string{"Ahad", "Isnin", "Selasa", "Rabu", "Khamis", "Jumaat", "Sabtu"}

	// Check hari tutup
	for _, h := range hariTutupList {
		if h == hariNum {
			return KedaiStatus{
				Status:    "TUTUP",
				Icon:      "🔴",
				Sebab:     fmt.Sprintf("Hari %s — tutup.", hariNames[hariNum]),
				MemoOwner: memo,
			}
		}
	}

	if bukaMinutes <= currentMinutes && currentMinutes < tutupMinutes {
		baki := tutupMinutes - currentMinutes
		jam := baki / 60
		minit := baki % 60
		return KedaiStatus{
			Status:    "BUKA",
			Icon:      "🟢",
			Sebab:     fmt.Sprintf("Beroperasi. Tutup %dj %dm lagi.", jam, minit),
			MemoOwner: memo,
		}
	}

	if currentMinutes < bukaMinutes {
		baki := bukaMinutes - currentMinutes
		jam := baki / 60
		minit := baki % 60
		return KedaiStatus{
			Status:    "TUTUP",
			Icon:      "🔴",
			Sebab:     fmt.Sprintf("Belum buka. Buka %dj %dm lagi.", jam, minit),
			MemoOwner: memo,
		}
	}

	return KedaiStatus{Status: "TUTUP", Icon: "🔴", Sebab: "Sudah tutup.", MemoOwner: memo}
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
		Timestamp: currentTime,
		Message:   message,
	})

	// Keep only last N messages
	if len(s.userMessageHistory[userID]) > s.maxHistoryPerUser {
		s.userMessageHistory[userID] = s.userMessageHistory[userID][len(s.userMessageHistory[userID])-s.maxHistoryPerUser:]
	}

	// CHECK 1: Duplicate message (last 5)
	history := s.userMessageHistory[userID]
	recentCount := 0
	startIdx := len(history) - 5
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < len(history); i++ {
		if history[i].Message == message {
			recentCount++
		}
	}
	if recentCount >= 2 {
		return true, "🚫 *Jangan spam mesej yang sama!* 😅"
	}

	// CHECK 2: Too many messages in 10 seconds
	count10s := 0
	for _, msg := range history {
		if currentTime.Sub(msg.Timestamp).Seconds() < 10 {
			count10s++
		}
	}
	if count10s >= 5 {
		return true, "⏳ *Slow slow boss! Tunggu sekejap...* 😄"
	}

	// CHECK 3: Too many messages in 1 minute
	count60s := 0
	for _, msg := range history {
		if currentTime.Sub(msg.Timestamp).Seconds() < 60 {
			count60s++
		}
	}
	if count60s >= 15 {
		return true, "🚦 *Banyak sangat mesej! Rehat sekejap.* 😅"
	}

	return false, ""
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

	// Build menu list
	var menuList strings.Builder
	for _, m := range menuItems {
		nama, _ := m["nama"].(string)
		desc, _ := m["desc"].(string)
		harga, _ := m["harga"].(float64)
		menuList.WriteString(fmt.Sprintf("- **%s** — *%s* — `RM%.2f`\n", nama, desc, harga))
	}

	// Build hari tutup
	hariTutupList := getIntArray(hours, "hari_tutup")
	hariNames := []string{"Ahad", "Isnin", "Selasa", "Rabu", "Khamis", "Jumaat", "Sabtu"}
	var hariTutupNames []string
	for _, d := range hariTutupList {
		if d >= 0 && d <= 6 {
			hariTutupNames = append(hariTutupNames, hariNames[d])
		}
	}
	hariTutupStr := "Tiada"
	if len(hariTutupNames) > 0 {
		hariTutupStr = strings.Join(hariTutupNames, ", ")
	}

	memoOwner, _ := owner["memo"].(string)
	if memoOwner == "" {
		memoOwner = "Tiada"
	}

	// Build wajib & larangan lists
	var wajibList, laranganList strings.Builder
	for _, w := range s.prompt.Wajib {
		wajibList.WriteString("- " + w + "\n")
	}
	for _, l := range s.prompt.Larangan {
		laranganList.WriteString("- " + l + "\n")
	}

	statusIcon := status.Icon
	if statusIcon == "" {
		statusIcon = "🟢"
	}

	return fmt.Sprintf(`Anda adalah %s, %s.
Persona: %s Melayu %d tahun, %s.

⏰ Sekarang: %s
%s Status: **%s** — %s
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
		statusIcon, status.Status, status.Sebab,
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
// 💬 CHAT - REST API TO GEMINI
// ==============================================

func (s *SitiAI) Chat(userMessage, sessionID string) (string, error) {
	// SECURITY CHECK 1: Message length limit
	const maxInput = 500
	if len(userMessage) > maxInput {
		return fmt.Sprintf("⚠️ *Maaf, mesej terlalu panjang!* Sila ringkaskan kepada %d aksara.", maxInput), nil
	}

	log.Printf("[SitiAI] Processing message: %s", truncate(userMessage, 50))

	// Build conversation history (last 10 messages)
	var contents []map[string]interface{}

	// Add system prompt
	contents = append(contents, map[string]interface{}{
		"role": "user",
		"parts": []map[string]interface{}{
			{"text": s.GetSystemPrompt()},
		},
	})

	// Add conversation history from Firestore/session
	// For simplicity, we'll just use the current message
	contents = append(contents, map[string]interface{}{
		"role": "user",
		"parts": []map[string]interface{}{
			{"text": userMessage},
		},
	})

	// Build request payload
	requestPayload := map[string]interface{}{
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"topP":          0.45,
			"temperature":   0.7,
			"thinkingConfig": map[string]interface{}{
				"thinkingLevel": "MINIMAL",
			},
		},
	}

	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call Gemini API via REST
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not set")
	}

	modelID := "gemma-4-26b-a4b-it"
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?key=%s&alt=sse", modelID, apiKey)

	log.Printf("[SitiAI] Calling Gemini API...")

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Read SSE response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse SSE chunks
	var responseText strings.Builder
	lines := strings.Split(string(bodyBytes), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			dataStr := strings.TrimPrefix(line, "data: ")
			if dataStr == "[DONE]" {
				break
			}

			var chunk map[string]interface{}
			if err := json.Unmarshal([]byte(dataStr), &chunk); err != nil {
				continue
			}

			if candidates, ok := chunk["candidates"].([]interface{}); ok && len(candidates) > 0 {
				if candidate, ok := candidates[0].(map[string]interface{}); ok {
					if content, ok := candidate["content"].(map[string]interface{}); ok {
						if parts, ok := content["parts"].([]interface{}); ok {
							for _, part := range parts {
								if p, ok := part.(map[string]interface{}); ok {
									if text, ok := p["text"].(string); ok {
										responseText.WriteString(text)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	response := responseText.String()
	log.Printf("[SitiAI] Response generated: %d chars", len(response))
	return response, nil
}

// ==============================================
// 🛠️ HELPER FUNCTIONS
// ==============================================

func parseTimeToMinutes(timeStr string) int {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0
	}
	var hour, minute int
	fmt.Sscanf(parts[0], "%d", &hour)
	fmt.Sscanf(parts[1], "%d", &minute)
	return hour*60 + minute
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getIntArray(m map[string]interface{}, key string) []int {
	if v, ok := m[key]; ok {
		if arr, ok := v.([]interface{}); ok {
			var result []int
			for _, item := range arr {
				switch val := item.(type) {
				case float64:
					result = append(result, int(val))
				case int:
					result = append(result, val)
				case int64:
					result = append(result, int(val))
				}
			}
			return result
		}
	}
	return []int{}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
