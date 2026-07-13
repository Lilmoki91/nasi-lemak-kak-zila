```markdown
# 🍽️ Nasi Lemak Kak Zila

![Nasi Lemak Kak Zila](https://i.postimg.cc/nhVCC9Pd/media.jpg)

> **Progressive Web App (PWA) untuk Pemesanan Nasi Lemak Kak Zila**  
> Boleh install di phone, offline support, mesra mobile.  
> Dilengkapi AI Chatbot (SITI AI) dengan Go + REST API + Dual Model Fallback.

---

## 📲 PWA - Boleh Install di Phone

![PWA Icon](https://raw.githubusercontent.com/Lilmoki91/nasi-lemak-kak-zila/refs/heads/main/nasi-lemak-icon-512.png)

| Platform | Cara Install |
|----------|--------------|
| **Android** | Buka site di Chrome → Popup "Install App" → Install |
| **iPhone** | Buka site di Safari → Share → Add to Home Screen |
| **Desktop** | Buka site di Chrome → Ikon ⊕ di address bar → Install |

---

## 🌐 Demo Langsung

🔗 **[nasi-lemak-kak-zila.pages.dev](https://nasi-lemak-kak-zila.pages.dev/)**

---

## 📱 Paparan Aplikasi

| Menu Utama | Kad Bisnes | Hubungi | Lokasi |
|:---:|:---:|:---:|:---:|
| ![Menu](https://i.postimg.cc/nhVCC9Pd/media.jpg) | ![Kad Depan](https://i.postimg.cc/vZ2gCR74/media-(2).jpg) | ![Hubungi](https://i.postimg.cc/nhVCC9Pd/media.jpg) | ![Lokasi](https://i.postimg.cc/nhVCC9Pd/media.jpg) |

| Kad Depan | Kad Belakang |
|:---:|:---:|
| ![Kad Depan](https://i.postimg.cc/vZ2gCR74/media-(2).jpg) | ![Kad Belakang](https://i.postimg.cc/Pxf94qbN/media(3).png) |

---

## 🛒 Fungsi Utama

- **🍽️ Menu Digital** — 4 item dengan harga (boleh tambah/edit/padam dari Panel Admin)
- **🛒 Sistem Troli** — Tambah, kurang, kira jumlah automatik
- **💬 Order WhatsApp** — Hantar pesanan terus ke WhatsApp (berfungsi dalam PWA)
- **💳 Kad Bisnes Digital** — Depan & belakang, boleh simpan & kongsi
- **📞 Hubungi** — WhatsApp & panggilan terus
- **📍 Lokasi** — Google Maps & Waze
- **📲 PWA** — Boleh install, offline support, auto update
- **🤖 SITI AI Chatbot** — AI pembantu 24/7 dengan memori perbualan
- **🔄 Dual Model AI** — Gemini Flash Lite + Gemma 4 (auto fallback)
- **🤲 AI Berdoa** — Setiap respons SITI AI diakhiri dengan doa keberkatan
- **🔐 Panel Admin** — Owner boleh kawal semua dari webapps

---

## 📋 Menu & Harga

| Item | Harga |
|------|-------|
| 🍗 Nasi Lemak Berlauk | RM5.00 |
| 🍚 Nasi Lemak Biasa | RM2.00 |
| 🥟 Kaaripuf | RM1.00 |
| 🍹 Air Balang | RM1.00 |

---

## 📞 Hubungi

| Platform | Nombor / Alamat |
|----------|-----------------|
| WhatsApp | [+60 11-1164 0776](https://wa.me/601111640776) |
| Telefon | [+60 11-1164 0776](tel:+601111640776) |
| Lokasi | PPR Sri Pantai Blok 102, Kuala Lumpur |

---

## 🕐 Waktu Operasi

| Hari | Masa |
|------|------|
| Jumaat - Rabu | 7:30 PM - 12:00 AM |
| Khamis | ❌ Tutup |

---

## 🚀 Teknologi

| Teknologi | Fungsi |
|-----------|--------|
| **HTML5** | Struktur halaman |
| **CSS3** | Styling & animasi |
| **JavaScript (Vanilla)** | Sistem troli, navigasi, localStorage |
| **Go (Golang)** | Backend AI Chatbot |
| **REST API Direct** | Panggilan ke Gemini API tanpa library |
| **Gemini Flash Lite** | Model AI utama (laju & murah) |
| **Gemma 4** | Model AI fallback (power & reliable) |
| **Firebase Firestore** | Database menu, waktu, memori, settings |
| **Firebase Storage** | Simpan gambar menu |
| **PWA** | Installable, offline, Service Worker |
| **Cloudflare Pages** | Hosting frontend percuma & pantas |
| **Render** | Hosting backend AI (Go) |

---

## 📁 Struktur Fail

```

nasi-lemak-kak-zila/
├── 📄 index.html                    # Frontend PWA
├── 📄 manifest.json                 # PWA manifest
├── 📄 sw.js                         # Service Worker
├── 📄 style.css                     # Styling
├── 📄 script.js                     # JavaScript frontend
├── 🖼️ nasi-lemak-icon-192.png       # Ikon PWA 192x192
├── 🖼️ nasi-lemak-icon-512.png       # Ikon PWA 512x512
├── 🧠 persona.json                  # Identiti SITI AI
├── 📋 prompt.json                   # Arahan & doa SITI AI
├── ⚡ main.go                       # Backend AI (Go)
├── 📦 go.mod                        # Go module
├── 🔒 go.sum                        # Go checksum
└── 📖 README.md                     # Dokumentasi

```

---

## 🤖 SITI AI Features

| Feature | Status |
|---------|:------:|
| AI Chatbot 24/7 | ✅ |
| Dual Model (Gemini Flash + Gemma 4) | ✅ |
| Auto Fallback | ✅ |
| Memory Perbualan (10 mesej) | ✅ |
| Anti-Spam | ✅ |
| Block Mesej Panjang | ✅ |
| Markdown Formatting | ✅ |
| Doa Keberkatan | ✅ 🤲 |
| Baca Menu dari Firebase | ✅ |
| Baca Waktu dari Firebase | ✅ |
| Baca Memo Owner | ✅ |

---

## 🔧 PWA Features

| Feature | Status |
|---------|:------:|
| Installable | ✅ |
| Offline Support | ✅ |
| Auto Update | ✅ |
| Auto Clear Cache | ✅ |
| Maskable Icons | ✅ |
| WhatsApp dalam PWA | ✅ |
| Google Maps | ✅ |
| Waze | ✅ |

---

## 📝 Lesen

Projek ini untuk kegunaan peribadi **Nasi Lemak Kak Zila**.  
Dibina dengan ❤️ untuk menyokong perniagaan kecil.

---

**Sekian, terima kasih 🥰**

---

🤲 *"Ya Allah, berkati perniagaan Kak Zila, murahkan rezeki, permudahkan urusan, dan jadikan setiap butir nasi lemak sebagai sumber keberkatan. Amin."*
```
