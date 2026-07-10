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
    "time"

    "cloud.google.com/go/firestore"
    firebase "firebase.google.com/go"
    "google.golang.org/api/option"
)

// ==============================================
// 🧠 SITI AI STRUCT
// ==============================================
type SitiAI struct {
    apiKey  string
    model   string
    baseURL string
    persona map[string]interface{}
    prompt  map[string]interface{}
    db      *firestore.Client
}

func (s *SitiAI) Init() {
    ctx := context.Background()

    s.apiKey = os.Getenv("GEMINI_API_KEY")
    if s.apiKey == "" {
        log.Fatal("GEMINI_API_KEY environment variable is required")
    }

    s.model = "gemma-4-26b-a4b-it"
    s.baseURL = "https://generativelanguage.googleapis.com/v1beta/models"

    // Load persona & prompt
    if personaData, err := os.ReadFile("persona.json"); err == nil {
        json.Unmarshal(personaData, &s.persona)
    } else {
        s.persona = make(map[string]interface{})
    }

    if promptData, err := os.ReadFile("prompt.json"); err == nil {
        json.Unmarshal(promptData, &s.prompt)
    } else {
        s.prompt = make(map[string]interface{})
    }

    // ==============================================
    // 🔥 INIT FIREBASE
    // ==============================================
    firebaseCreds := os.Getenv("FIREBASE_CREDENTIALS")
    if firebaseCreds == "" {
        log.Println("⚠️ FIREBASE_CREDENTIALS tidak ditemui")
        return
    }

    var creds map[string]interface{}
    if err := json.Unmarshal([]byte(firebaseCreds), &creds); err != nil {
        log.Printf("⚠️ Gagal parse FIREBASE_CREDENTIALS: %v", err)
        return
    }

    credsJSON, _ := json.Marshal(creds)
    opt := option.WithCredentialsJSON(credsJSON)

    app, err := firebase.NewApp(ctx, nil, opt)
    if err != nil {
        log.Printf("⚠️ Firebase App init gagal: %v", err)
        return
    }

    db, err := app.Firestore(ctx)
    if err != nil {
        log.Printf("⚠️ Firestore init gagal: %v", err)
        return
    }

    s.db = db
    log.Println("✅ Firebase siap!")
}

// ==============================================
// 📋 LOAD OWNER SETTINGS DARI FIREBASE
// ==============================================
func (s *SitiAI) LoadOwnerSettings() map[string]interface{} {
    if s.db == nil {
        return map[string]interface{}{}
    }

    ctx := context.Background()
    doc, err := s.db.Collection("settings").Doc("shop_settings").Get(ctx)
    if err != nil {
        return map[string]interface{}{}
    }

    return doc.Data()
}

// ==============================================
// ⏰ LOAD OPERATING HOURS DARI FIREBASE
// ==============================================
func (s *SitiAI) LoadOperatingHours() map[string]interface{} {
    if s.db == nil {
        return map[string]interface{}{}
    }

    ctx := context.Background()
    doc, err := s.db.Collection("settings").Doc("operating_hours").Get(ctx)
    if err != nil {
        return map[string]interface{}{}
    }

    return doc.Data()
}

// ==============================================
// 🍗 LOAD MENU DARI FIREBASE
// ==============================================
func (s *SitiAI) LoadMenu() []map[string]interface{} {
    if s.db == nil {
        return []map[string]interface{}{}
    }

    ctx := context.Background()
    docs, err := s.db.Collection("menu").Where("aktif", "==", true).Documents(ctx).GetAll()
    if err != nil {
        return []map[string]interface{}{}
    }

    menu := []map[string]interface{}{}
    for _, doc := range docs {
        d := doc.Data()
        menu = append(menu, map[string]interface{}{
            "nama":     d["nama"],
            "desc":     d["desc"],
            "harga":    d["harga"],
            "featured": d["featured"],
            "gambar":   d["gambar"],
        })
    }

    return menu
}

// ==============================================
// 🤖 PANGGIL GEMINI API (REST DIRECT)
// ==============================================
func (s *SitiAI) callGemini(prompt string) (string, error) {
    url := fmt.Sprintf("%s/%s:generateContent?key=%s", s.baseURL, s.model, s.apiKey)

    body := map[string]interface{}{
        "contents": []map[string]interface{}{
            {
                "role": "user",
                "parts": []map[string]string{{
                    "text": prompt,
                }},
            },
        },
        "generationConfig": map[string]interface{}{
            "temperature": 0.7,
            "topP":        0.45,
        },
    }

    jsonBody, _ := json.Marshal(body)
    resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    respBody, _ := io.ReadAll(resp.Body)

    var result map[string]interface{}
    if err := json.Unmarshal(respBody, &result); err != nil {
        return "", err
    }

    candidates, ok := result["candidates"].([]interface{})
    if !ok || len(candidates) == 0 {
        return "", fmt.Errorf("no candidates found in response")
    }

    candidate, ok := candidates[0].(map[string]interface{})
    if !ok {
        return "", fmt.Errorf("invalid candidate format")
    }

    content, ok := candidate["content"].(map[string]interface{})
    if !ok {
        return "", fmt.Errorf("invalid content format")
    }

    parts, ok := content["parts"].([]interface{})
    if !ok || len(parts) == 0 {
        return "", fmt.Errorf("no parts found in content")
    }

    part, ok := parts[0].(map[string]interface{})
    if !ok {
        return "", fmt.Errorf("invalid part format")
    }

    text, ok := part["text"].(string)
    if !ok {
        return "", fmt.Errorf("text field not found or not a string")
    }

    return text, nil
}

// ==============================================
// 🕐 MASA MALAYSIA
// ==============================================
func (s *SitiAI) GetMalaysiaTime() map[string]interface{} {
    loc, _ := time.LoadLocation("Asia/Kuala_Lumpur")
    now := time.Now().In(loc)

    hariMap := map[string]string{
        "Monday": "Isnin", "Tuesday": "Selasa", "Wednesday": "Rabu",
        "Thursday": "Khamis", "Friday": "Jumaat", "Saturday": "Sabtu", "Sunday": "Ahad",
    }

    return map[string]interface{}{
        "waktu_penuh": fmt.Sprintf("%s, %s", hariMap[now.Format("Monday")], now.Format("02/01/2006 03:04 PM")),
        "hari":        hariMap[now.Format("Monday")],
        "hari_num":    int(now.Weekday()),
        "jam_12h":     now.Format("03:04 PM"),
    }
}

// ==============================================
// 🟢 STATUS KEDAI
// ==============================================
func (s *SitiAI) CheckKedaiStatus() map[string]interface{} {
    owner := s.LoadOwnerSettings()
    hours := s.LoadOperatingHours()

    mode, _ := owner["mode"].(string)
    memo, _ := owner["memo"].(string)

    // Override mode
    if mode == "BUKA" {
        return map[string]interface{}{"status": "BUKA", "sebab": memo}
    }
    if mode == "TUTUP" {
        return map[string]interface{}{"status": "TUTUP", "sebab": memo}
    }

    // AUTO - guna waktu operasi
    bukaStr, _ := hours["buka"].(string)
    tutupStr, _ := hours["tutup"].(string)
    if bukaStr == "" || tutupStr == "" {
        return map[string]interface{}{"status": "TUTUP", "sebab": "Waktu belum ditetapkan."}
    }

    var bH, bM, tH, tM int
    fmt.Sscanf(bukaStr, "%d:%d", &bH, &bM)
    fmt.Sscanf(tutupStr, "%d:%d", &tH, &tM)
    buka := bH*60 + bM
    tutup := tH*60 + tM
    if tutup == 0 {
        tutup = 24 * 60
    }

    masa := s.GetMalaysiaTime()
    hariNum := masa["hari_num"].(int)
    now := time.Now().In(time.FixedZone("MYT", 8*60*60))
    current := now.Hour()*60 + now.Minute()

    // Hari tutup
    hariTutupList := []int{}
    if h, ok := hours["hari_tutup"].([]interface{}); ok {
        for _, v := range h {
            if day, ok := v.(float64); ok {
                hariTutupList = append(hariTutupList, int(day))
            }
        }
    }

    hariNames := []string{"Ahad", "Isnin", "Selasa", "Rabu", "Khamis", "Jumaat", "Sabtu"}

    for _, d := range hariTutupList {
        if hariNum == d {
            return map[string]interface{}{"status": "TUTUP", "sebab": fmt.Sprintf("Hari %s — tutup.", hariNames[hariNum])}
        }
    }

    if buka <= current && current < tutup {
        baki := tutup - current
        j, m := baki/60, baki%60
        return map[string]interface{}{"status": "BUKA", "sebab": fmt.Sprintf("Beroperasi. Tutup %dj %dm lagi.", j, m)}
    }

    if current < buka {
        baki := buka - current
        j, m := baki/60, baki%60
        return map[string]interface{}{"status": "TUTUP", "sebab": fmt.Sprintf("Belum buka. Buka %dj %dm lagi.", j, m)}
    }

    return map[string]interface{}{"status": "TUTUP", "sebab": "Sudah tutup."}
}

// ==============================================
// 📝 SYSTEM PROMPT (DENGAN DATA FIREBASE)
// ==============================================
func (s *SitiAI) GetSystemPrompt() string {
    masa := s.GetMalaysiaTime()
    status := s.CheckKedaiStatus()
    menuItems := s.LoadMenu()
    owner := s.LoadOwnerSettings()
    hours := s.LoadOperatingHours()

    // Bina menu list
    menuList := []string{}
    for _, m := range menuItems {
        nama, _ := m["nama"].(string)
        desc, _ := m["desc"].(string)
        harga, _ := m["harga"].(float64)
        
        menuList = append(menuList, fmt.Sprintf("- **%s** — *%s* — `RM%.2f`", nama, desc, harga))
    }

    // Hari tutup
    hariTutupList := []int{}
    if h, ok := hours["hari_tutup"].([]interface{}); ok {
        for _, v := range h {
            if day, ok := v.(float64); ok {
                hariTutupList = append(hariTutupList, int(day))
            }
        }
    }

    hariNames := []string{"Ahad", "Isnin", "Selasa", "Rabu", "Khamis", "Jumaat", "Sabtu"}
    hariTutupNames := []string{}
    for _, d := range hariTutupList {
        if d >= 0 && d < 7 {
            hariTutupNames = append(hariTutupNames, hariNames[d])
        }
    }

    memo := "Tiada"
    if m, ok := owner["memo"].(string); ok && m != "" {
        memo = m
    }

    return fmt.Sprintf(`Anda adalah SITI AI, pembantu Zila Food.
Persona: Wanita Melayu, mesra, sopan, ceria.

⏰ Sekarang: %v
🟢 Status: **%v** — %v
📝 Memo Owner: %v

📍 PPR Sri Pantai Blok 102, Kuala Lumpur
🗺️ [Google Maps](https://maps.google.com/?q=PPR+Sri+Pantai+Blok+102)
📲 [WhatsApp](https://wa.me/601111640776)
📅 Hari Tutup: %v

🍗 MENU:
%v

Format WAJIB Markdown:
- ## untuk tajuk
- **bold** untuk nama menu
- *italic* untuk deskripsi
- `+"`RM5`"+` untuk harga
- [teks](url) untuk link

Senang je! 😊`,
        masa["waktu_penuh"],
        status["status"], status["sebab"],
        memo,
        strings.Join(hariTutupNames, ", "),
        strings.Join(menuList, "\n"),
    )
}

// ==============================================
// 💬 CHAT
// ==============================================
func (s *SitiAI) Chat(message string) string {
    fullPrompt := s.GetSystemPrompt() + "\n\nUser: " + message
    response, err := s.callGemini(fullPrompt)
    if err != nil {
        log.Printf("Error Gemini: %v", err)
        return "Maaf, SITI AI mengalami gangguan."
    }
    return response
}

// ==============================================
// 🌐 HTTP SERVER
// ==============================================
func main() {
    siti := &SitiAI{}
    siti.Init()

    http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        var req map[string]string
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
            return
        }

        message := req["message"]
        response := siti.Chat(message)

        json.NewEncoder(w).Encode(map[string]string{"response": response})
    })

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "name": "SITI AI", "status": "running",
        })
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("🤖 SITI AI Go + Firebase running on :%s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
