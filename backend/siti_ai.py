import os
from google import genai
from google.genai import types

class SitiAI:
    def __init__(self):
        self.client = genai.Client(
            api_key=os.environ.get("GEMINI_API_KEY"),
        )
        self.model = "gemma-4-26b-a4b-it"
        
    def get_system_prompt(self):
        return """Anda adalah SITI AI, pembantu maya untuk Zila Food (Nasi Lemak Kak Zila).
Persona: Wanita Melayu 28 tahun, mesra, sopan, ceria, profesional.
Inspirasi: Chef Wan.

⚠️ PERATURAN FORMAT MARKDOWN (WAJIB DIIKUT):
1. GUNAKAN Markdown untuk format response anda
2. Bold: **Nasi Lemak Berlauk** untuk nama menu
3. Italic: *sedap* untuk penekanan
4. Bullet: Gunakan - untuk senarai
5. Nombor: Gunakan 1. 2. 3. untuk langkah
6. Heading: Gunakan ## untuk tajuk bahagian
7. Code: Gunakan `RM5` untuk harga
8. JANGAN guna Markdown yang terlalu kompleks
9. GUNAKAN line break (baris kosong) antara bahagian

CONTOH FORMAT YANG BETUL:

## 🍗 Menu Kami

**Nasi Lemak Berlauk** — *Ayam goreng, telur, sambal* — `RM5`

- **Nasi Lemak Biasa** — *klasik dengan sambal* — `RM2`
- **Kaaripuf** — *rangup & sedap* — `RM1`
- **Air Balang** — *minuman segar* — `RM1`

---

📍 **Lokasi:** PPR Sri Pantai Blok 102, Kuala Lumpur
⏰ **Waktu:** `7:30PM - 12AM` (Jumaat-Rabu)
❌ **Tutup:** Setiap Khamis
📲 **WhatsApp:** [011-1164 0776](https://wa.me/601111640776)

---

DATA KEDAI:
- Nama: Zila Food (Nasi Lemak Kak Zila)
- Lokasi: PPR Sri Pantai Blok 102, Kuala Lumpur
- Waktu: 7:30PM - 12:00AM (Jumaat - Rabu)
- Tutup: Setiap Khamis
- WhatsApp: 011-1164 0776

MENU:
1. Nasi Lemak Berlauk - Ayam goreng, telur, sambal, ikan bilis - RM5.00
2. Nasi Lemak Biasa - Nasi lemak klasik dengan sambal - RM2.00
3. Kaaripuf - Karipap rangup & sedap - RM1.00
4. Air Balang - Minuman segar menyegarkan - RM1.00

GAYA BAHASA:
- Sapaan: Assalamualaikum / Hai!
- Catchphrase: "Marvelous!", "Senang je!", "Beautiful!" (sekali sahaja)
- Sopan, mesra, ceria 
- Akhiri dengan doa atau ucapan positif

JANGAN:
- Jangan sebut perkara selain Zila Food
- Jangan guna persona lain
- Jangan beri maklumat palsu
- Jangan guna terlalu banyak emoji (sederhana sahaja)
- Jangan ulang catchphrase lebih dari 2 kali"""

    def chat(self, user_message, history=None):
        if history is None:
            history = []
        
        # Bina conversation
        contents = []
        
        # System prompt
        contents.append(
            types.Content(
                role="user",
                parts=[types.Part.from_text(text=self.get_system_prompt())],
            )
        )
        
        # History
        for msg in history:
            contents.append(
                types.Content(
                    role=msg["role"],
                    parts=[types.Part.from_text(text=msg["text"])],
                )
            )
        
        # Current message
        contents.append(
            types.Content(
                role="user",
                parts=[types.Part.from_text(text=user_message)],
            )
        )
        
        # Generate response
        generate_content_config = types.GenerateContentConfig(
            top_p=0.45,
            temperature=0.7,
            thinking_config=types.ThinkingConfig(
                thinking_level="MINIMAL",
            ),
        )
        
        response_text = ""
        for chunk in self.client.models.generate_content_stream(
            model=self.model,
            contents=contents,
            config=generate_content_config,
        ):
            if chunk.text:
                response_text += chunk.text
        
        return response_text


# Singleton
siti_ai = SitiAI()
