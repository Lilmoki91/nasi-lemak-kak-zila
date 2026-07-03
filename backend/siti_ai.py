import os
import json
from datetime import datetime, timezone, timedelta
from google import genai
from google.genai import types
import firebase_admin
from firebase_admin import credentials, firestore

class SitiAI:
    def __init__(self):
        self.client = genai.Client(
            api_key=os.environ.get("GEMINI_API_KEY"),
        )
        self.model = "gemma-4-26b-a4b-it"
        
        # Load persona & prompt dari JSON
        with open('persona.json', 'r', encoding='utf-8') as f:
            self.persona = json.load(f)
        with open('prompt.json', 'r', encoding='utf-8') as f:
            self.prompt = json.load(f)

        # 🔥 INIT FIREBASE DARI .env
        if not firebase_admin._apps:
            firebase_creds = json.loads(os.environ.get("FIREBASE_CREDENTIALS", "{}"))
            cred = credentials.Certificate(firebase_creds)
            firebase_admin.initialize_app(cred)
        self.db = firestore.client()
        print('✅ Firebase siap!')

    # ==============================================
    # 🕐 MASA MALAYSIA (GMT+8)
    # ==============================================
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

    # ==============================================
    # 📋 LOAD OWNER SETTINGS (DARI FIREBASE)
    # ==============================================
    def load_owner_settings(self):
        """Baca settings kedai dari Firebase"""
        try:
            doc = self.db.collection("settings").document("kedai").get()
            if doc.exists:
                return doc.to_dict()
        except:
            pass
        # Fallback default
        return {
            "mode": "AUTO",
            "memo": "",
            "waktu_buka": "19:30",
            "waktu_tutup": "00:00",
            "hari_tutup": [4]  # Default: Khamis (index 4)
        }

    # ==============================================
    # 🍗 LOAD MENU (DARI FIREBASE)
    # ==============================================
    def load_menu(self):
        """Baca menu dari Firebase"""
        try:
            docs = self.db.collection("menu").where("aktif", "==", True).stream()
            menu = []
            for doc in docs:
                data = doc.to_dict()
                menu.append({
                    "nama": data.get("nama", ""),
                    "desc": data.get("desc", ""),
                    "harga": data.get("harga", 0),
                    "featured": data.get("featured", False)
                })
            return menu if menu else self.persona.get('menu', [])
        except:
            return self.persona.get('menu', [])

    # ==============================================
    # 🟢 STATUS KEDAI (BACA WAKTU DARI FIREBASE)
    # ==============================================
    def check_kedai_status(self):
        owner = self.load_owner_settings()
        mode = owner.get("mode", "AUTO")
        memo = owner.get("memo", "")
        
        # 🔥 BACA WAKTU DARI FIREBASE (bukan hardcode!)
        waktu_buka_str = owner.get("waktu_buka", "19:30")
        waktu_tutup_str = owner.get("waktu_tutup", "00:00")
        hari_tutup_list = owner.get("hari_tutup", [4])
        
        # Parse waktu
        try:
            buka_parts = waktu_buka_str.split(":")
            tutup_parts = waktu_tutup_str.split(":")
            buka = int(buka_parts[0]) * 60 + int(buka_parts[1])
            tutup = int(tutup_parts[0]) * 60 + int(tutup_parts[1])
            if tutup == 0:
                tutup = 24 * 60  # 00:00 = 1440 minit
        except:
            buka = 19 * 60 + 30  # Fallback: 7:30 PM
            tutup = 24 * 60      # Fallback: 12:00 AM
        
        # Format waktu untuk display
        def format_waktu(menit):
            jam = menit // 60
            minit = menit % 60
            if menit >= 24 * 60:
                return "12:00 AM"
            elif jam >= 12:
                return f"{jam-12 if jam>12 else jam}:{minit:02d} PM"
            else:
                return f"{jam}:{minit:02d} AM"
        
        waktu_buka_display = format_waktu(buka)
        waktu_tutup_display = format_waktu(tutup)
        
        # Nama hari dalam BM
        hari_names = ["Ahad", "Isnin", "Selasa", "Rabu", "Khamis", "Jumaat", "Sabtu"]
        hari_tutup_names = [hari_names[d] for d in hari_tutup_list if 0 <= d <= 6]
        
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
        
        # AUTO
        masa = self.get_malaysia_time()
        hari_num = masa["hari_num"]
        now = datetime.now(timezone(timedelta(hours=8)))
        current_minutes = now.hour * 60 + now.minute
        
        # Check hari tutup
        if hari_num in hari_tutup_list:
            hari_tutup_str = ", ".join(hari_tutup_names)
            next_hari = (hari_num + 1) % 7
            while next_hari in hari_tutup_list:
                next_hari = (next_hari + 1) % 7
            return {
                "status": "TUTUP",
                "sebab": f"Hari {hari_names[hari_num]} — kedai tutup",
                "next_buka": f"{hari_names[next_hari]}, {waktu_buka_display}"
            }
        
        # Check waktu operasi
        if current_minutes >= buka and current_minutes < tutup:
            baki = tutup - current_minutes
            jam, minit = divmod(baki, 60)
            return {
                "status": "BUKA",
                "sebab": f"Kedai sedang beroperasi. Tutup dalam {jam}j {minit}m lagi.",
                "baki": f"{jam} jam {minit} minit"
            }
        elif current_minutes < buka:
            baki = buka - current_minutes
            jam, minit = divmod(baki, 60)
            return {
                "status": "TUTUP",
                "sebab": f"Kedai belum dibuka. Akan dibuka dalam {jam}j {minit}m lagi.",
                "next_buka": f"Hari ini, {waktu_buka_display} (dalam {jam} jam {minit} minit)"
            }
        else:
            next_hari = (hari_num + 1) % 7
            while next_hari in hari_tutup_list:
                next_hari = (next_hari + 1) % 7
            return {
                "status": "TUTUP",
                "sebab": "Kedai sudah tutup untuk hari ini.",
                "next_buka": f"{hari_names[next_hari]}, {waktu_buka_display}"
            }

    # ==============================================
    # 📝 SYSTEM PROMPT LENGKAP
    # ==============================================
    def get_system_prompt(self):
        masa = self.get_malaysia_time()
        status = self.check_kedai_status()
        menu_items = self.load_menu()
        owner = self.load_owner_settings()
        
        # Bina senarai menu dari Firebase
        menu_list = "\n".join(
            [f"- **{m['nama']}** — *{m['desc']}* — `RM{m['harga']:.2f}`" for m in menu_items]
        )
        
        # Dapatkan info hari tutup
        hari_tutup_list = owner.get("hari_tutup", [4])
        hari_names = ["Ahad", "Isnin", "Selasa", "Rabu", "Khamis", "Jumaat", "Sabtu"]
        hari_tutup_names = [hari_names[d] for d in hari_tutup_list if 0 <= d <= 6]
        
        return f"""Anda adalah {self.persona['watak']['nama']}, {self.persona['watak']['peranan']}.
Persona: {self.persona['watak']['jantina']} Melayu {self.persona['watak']['umur']} tahun, {', '.join(self.persona['watak']['gaya'])}.

⏰ **DATA MASA TERKINI:**
- Sekarang: {masa['waktu_penuh']}
- Status Kedai: **{status['status']}**
- Sebab: {status['sebab']}

📍 **DATA KEDAI:**
- Nama: {self.persona['kedai']['nama']}
- Lokasi: {self.persona['kedai']['lokasi']}
- Google Maps: {self.persona['kedai']['google_maps']}
- Waze: {self.persona['kedai']['waze']}
- WhatsApp: {self.persona['kedai']['whatsapp']}
- Hari Tutup: {', '.join(hari_tutup_names)}

🍗 **MENU:**
{menu_list}

🎤 **GAYA BAHASA:**
- Sapaan: {self.persona['watak']['sapaan']}
- Catchphrase: {', '.join(self.persona['watak']['catchphrase'])}

📋 **FORMAT MARKDOWN (WAJIB):**
- Bold: **nama menu** | Italic: *sedap* | Code: `RM5` | Bullet: - | Link: [teks](url)

✅ **WAJIB:**
{chr(10).join(['- ' + x for x in self.prompt['wajib']])}

🚫 **LARANGAN:**
{chr(10).join(['- ' + x for x in self.prompt['larangan']])}"""

    # ==============================================
    # 💬 CHAT FUNCTION
    # ==============================================
    def chat(self, user_message, history=None):
        MAX_INPUT = 500
        
        if len(user_message) > MAX_INPUT:
            return f"⚠️ *Maaf, mesej terlalu panjang!* Sila ringkaskan kepada {MAX_INPUT} aksara."
        
        if history:
            last_msgs = [msg["text"] for msg in history if msg["role"] == "user"]
            if last_msgs and last_msgs[-1] == user_message:
                return "🤔 *Anda dah hantar mesej yang sama.* Ada soalan lain?"
        
        if history is None:
            history = []
        
        contents = [
            types.Content(role="user", parts=[types.Part.from_text(text=self.get_system_prompt())])
        ]
        for msg in history:
            contents.append(types.Content(role=msg["role"], parts=[types.Part.from_text(text=msg["text"])]))
        contents.append(types.Content(role="user", parts=[types.Part.from_text(text=user_message)]))
        
        config = types.GenerateContentConfig(
            top_p=0.45, temperature=0.7,
            thinking_config=types.ThinkingConfig(thinking_level="MINIMAL"),
        )
        
        response_text = ""
        for chunk in self.client.models.generate_content_stream(
            model=self.model, contents=contents, config=config
        ):
            if chunk.text:
                response_text += chunk.text
        
        return response_text


# Singleton
siti_ai = SitiAI()
