import os
import json
from datetime import datetime, timezone, timedelta
from google import genai
from google.genai import types

class SitiAI:
    def __init__(self):
        self.client = genai.Client(
            api_key=os.environ.get("GEMINI_API_KEY"),
        )
        self.model = "gemma-4-26b-a4b-it"
        
        with open('persona.json', 'r', encoding='utf-8') as f:
            self.persona = json.load(f)
        with open('prompt.json', 'r', encoding='utf-8') as f:
            self.prompt = json.load(f)

    def get_malaysia_time(self):
        tz = timezone(timedelta(hours=8))
        now = datetime.now(tz)
    
        hari_map = {
            "Monday": "Isnin", "Tuesday": "Selasa", "Wednesday": "Rabu",
            "Thursday": "Khamis", "Friday": "Jumaat", "Saturday": "Sabtu", "Sunday": "Ahad"
        }
    
        bulan_list = ["", "Januari", "Februari", "Mac", "April", "Mei", "Jun",
                      "Julai", "Ogos", "September", "Oktober", "November", "Disember"]
        hari_english = now.strftime("%A")
        hari_bm = hari_map[hari_english]
    
        return {
            "jam_12h": now.strftime("%I:%M %p"),
            "jam_24h": now.strftime("%H:%M"),
            "hari": hari_bm,
            "hari_english": hari_english,
            "hari_num": now.weekday(),
            "tarikh_penuh": f"{now.day} {bulan_list[now.month]} {now.year}",
            "waktu_penuh": f"{hari_bm}, {now.day} {bulan_list[now.month]} {now.year}, {now.strftime('%I:%M %p')}"
        }

    def load_owner_settings(self):
        try:
            with open('owner_settings.json', 'r', encoding='utf-8') as f:
                return json.load(f)
        except:
            return {"mode": "AUTO", "memo": ""}

    def check_kedai_status(self):
        # 🔥 CHECK OWNER OVERRIDE DULU!
        owner = self.load_owner_settings()
        mode = owner.get("mode", "AUTO")
        memo = owner.get("memo", "")
        
        if mode == "BUKA":
            return {
                "status": "BUKA",
                "sebab": memo if memo else "Kedai dibuka khas oleh owner! 🟢",
                "override": True,
                "memo_owner": memo
            }
        elif mode == "TUTUP":
            return {
                "status": "TUTUP",
                "sebab": memo if memo else "Kedai ditutup oleh owner. 🔴",
                "override": True,
                "memo_owner": memo
            }
        
        # 🔥 AUTO — guna jadual biasa
        masa = self.get_malaysia_time()
        hari_num = masa["hari_num"]
        now = datetime.now(timezone(timedelta(hours=8)))
        current_minutes = now.hour * 60 + now.minute
        
        buka = 19 * 60 + 30
        tutup = 24 * 60
        
        if hari_num == 4:
            result = {
                "status": "TUTUP",
                "sebab": "Hari Khamis — kedai tutup sepanjang hari",
                "next_buka": "Esok Jumaat, 7:30 PM"
            }
        elif current_minutes >= buka and current_minutes < tutup:
            baki = tutup - current_minutes
            jam = baki // 60
            minit = baki % 60
            result = {
                "status": "BUKA",
                "sebab": f"Kedai sedang beroperasi. Tutup dalam {jam}j {minit}m lagi.",
                "baki": f"{jam} jam {minit} minit"
            }
        elif current_minutes < buka:
            baki = buka - current_minutes
            jam = baki // 60
            minit = baki % 60
            result = {
                "status": "TUTUP",
                "sebab": f"Kedai belum dibuka. Akan dibuka dalam {jam}j {minit}m lagi.",
                "next_buka": f"Hari ini, 7:30 PM (dalam {jam} jam {minit} minit)"
            }
        else:
            result = {
                "status": "TUTUP",
                "sebab": "Kedai sudah tutup untuk hari ini.",
                "next_buka": "Esok, 7:30 PM"
            }
        
        # 🔥 TAMBAH MEMO OWNER (walaupun AUTO)
        if memo:
            result["memo_owner"] = memo
            result["sebab"] = result["sebab"] + f" 📝 Notis: {memo}"
        
        return result

    def get_system_prompt(self):
        masa = self.get_malaysia_time()
        status = self.check_kedai_status()
        
        menu_list = ""
        for item in self.persona['menu']:
            menu_list += f"- **{item['nama']}** — *{item['desc']}* — `RM{item['harga']:.2f}`\n"
        
        return f"""Anda adalah {self.persona['watak']['nama']}, {self.persona['watak']['peranan']}.
Persona: {self.persona['watak']['jantina']} Melayu {self.persona['watak']['umur']} tahun, {', '.join(self.persona['watak']['gaya'])}.
Inspirasi: {self.persona['watak']['inspirasi']}.
Identiti: {self.persona['watak']['identiti']}

⏰ **DATA MASA TERKINI (WAJIB GUNA):**
- Sekarang: {masa['waktu_penuh']}
- Hari: {masa['hari']}
- Tarikh: {masa['tarikh_penuh']}
- Status Kedai: **{status['status']}**
- Sebab: {status['sebab']}

📍 **DATA KEDAI:**
- Nama: {self.persona['kedai']['nama']}
- Lokasi: {self.persona['kedai']['lokasi']}
- Google Maps: {self.persona['kedai']['google_maps']}
- Waze: {self.persona['kedai']['waze']}
- WhatsApp: {self.persona['kedai']['whatsapp']}
- Waktu Operasi: {self.persona['kedai']['waktu_buka']} - {self.persona['kedai']['waktu_tutup']}
- Hari Operasi: {', '.join(self.persona['kedai']['hari_operasi'])}
- Hari Tutup: {', '.join(self.persona['kedai']['hari_tutup'])}

🍗 **MENU:**
{menu_list}

🎤 **GAYA BAHASA:**
- Sapaan: {self.persona['watak']['sapaan']}
- Catchphrase: {', '.join(self.persona['watak']['catchphrase'])}
- Nada: {self.persona['watak']['gaya'][0]}, {self.persona['watak']['gaya'][1]}

📋 **FORMAT MARKDOWN (WAJIB):**
- Bold: **nama menu** untuk nama menu
- Italic: *sedap* untuk penekanan
- Code: `RM5` untuk harga
- Bullet: - untuk senarai
- Heading: ## untuk tajuk
- Link: [teks](url) untuk pautan

✅ **WAJIB:**
{chr(10).join(['- ' + x for x in self.prompt['wajib']])}

🚫 **LARANGAN:**
{chr(10).join(['- ' + x for x in self.prompt['larangan']])}"""

    def chat(self, user_message, history=None):
        MAX_INPUT = 500
        
        if len(user_message) > MAX_INPUT:
            return f"⚠️ *Maaf, mesej terlalu panjang!* Sila ringkaskan kepada {MAX_INPUT} aksara. Senang je! 😊"
        
        if history:
            last_user_msgs = [msg["text"] for msg in history if msg["role"] == "user"]
            if last_user_msgs and last_user_msgs[-1] == user_message:
                return "🤔 *Anda dah hantar mesej yang sama.* Ada soalan lain yang Siti boleh bantu? 😊"
        
        if history is None:
            history = []
        
        contents = []
        
        contents.append(
            types.Content(
                role="user",
                parts=[types.Part.from_text(text=self.get_system_prompt())],
            )
        )
        
        for msg in history:
            contents.append(
                types.Content(
                    role=msg["role"],
                    parts=[types.Part.from_text(text=msg["text"])],
                )
            )
        
        contents.append(
            types.Content(
                role="user",
                parts=[types.Part.from_text(text=user_message)],
            )
        )
        
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
