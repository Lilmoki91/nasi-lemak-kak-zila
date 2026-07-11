import os
import json
import time
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
        
        with open('persona.json', 'r', encoding='utf-8') as f:
            self.persona = json.load(f)
        with open('prompt.json', 'r', encoding='utf-8') as f:
            self.prompt = json.load(f)

        if not firebase_admin._apps:
            firebase_creds = json.loads(os.environ.get("FIREBASE_CREDENTIALS", "{}"))
            cred = credentials.Certificate(firebase_creds)
            firebase_admin.initialize_app(cred)
        self.db = firebase_admin.firestore.client()  # ✅ BETUL: guna firebase_admin
        
        # 🔥 ANTI-SPAM: Track user messages
        self.user_message_history = {}
        self.MAX_HISTORY_PER_USER = 20

    def get_malaysia_time(self):
        tz = timezone(timedelta(hours=8))
        now = datetime.now(tz)
        hari_map = {"Monday":"Isnin","Tuesday":"Selasa","Wednesday":"Rabu","Thursday":"Khamis","Friday":"Jumaat","Saturday":"Sabtu","Sunday":"Ahad"}
        bulan_list = ["","Januari","Februari","Mac","April","Mei","Jun","Julai","Ogos","September","Oktober","November","Disember"]
        return {
            "jam_12h": now.strftime("%I:%M %p"),
            "jam_24h": now.strftime("%H:%M"),
            "hari": hari_map[now.strftime("%A")],
            "hari_num": now.weekday(),
            "tarikh_penuh": f"{now.day} {bulan_list[now.month]} {now.year}",
            "waktu_penuh": f"{hari_map[now.strftime('%A')]}, {now.day} {bulan_list[now.month]} {now.year}, {now.strftime('%I:%M %p')}"
        }

    def load_owner_settings(self):
        try:
            doc = self.db.collection("settings").document("shop_settings").get()
            if doc.exists:
                return doc.to_dict()
        except Exception as e:
            print(f"[SitiAI] Error load owner settings: {e}")
        return {}

    def load_operating_hours(self):
        try:
            doc = self.db.collection("settings").document("operating_hours").get()
            if doc.exists:
                return doc.to_dict()
        except Exception as e:
            print(f"[SitiAI] Error load operating hours: {e}")
        return {}

    def load_menu(self):
        try:
            docs = self.db.collection("menu").where("aktif","==",True).stream()
            menu = []
            for doc in docs:
                d = doc.to_dict()
                menu.append({"nama":d.get("nama",""),"desc":d.get("desc",""),"harga":d.get("harga",0),"featured":d.get("featured",False)})
            return menu
        except Exception as e:
            print(f"[SitiAI] Error load menu: {e}")
        return []

    # ==============================================
    # 🔒 ANTI-SPAM: CHECK MESSAGE RATE
    # ==============================================
    def check_spam(self, user_id, message):
        current_time = time.time()
        
        if user_id not in self.user_message_history:
            self.user_message_history[user_id] = []
        
        self.user_message_history[user_id].append((current_time, message))
        
        if len(self.user_message_history[user_id]) > self.MAX_HISTORY_PER_USER:
            self.user_message_history[user_id] = self.user_message_history[user_id][-self.MAX_HISTORY_PER_USER:]
        
        # 🔥 CHECK 1: Duplicate message (last 5 messages)
        recent_msgs = [msg for _, msg in self.user_message_history[user_id][-5:]]
        if recent_msgs.count(message) >= 2:
            return True, " *Jangan spam mesej yang sama!* "
        
        # 🔥 CHECK 2: Too many messages in 10 seconds
        recent_10s = [t for t, _ in self.user_message_history[user_id] if current_time - t < 10]
        if len(recent_10s) >= 5:
            return True, "⏳ *Slow slow boss! Tunggu sekejap...* 😄"
        
        # 🔥 CHECK 3: Too many messages in 1 minute
        recent_60s = [t for t, _ in self.user_message_history[user_id] if current_time - t < 60]
        if len(recent_60s) >= 15:
            return True, "🚦 *Banyak sangat mesej! Rehat sekejap.* 😅"
        
        return False, None

    def check_kedai_status(self):
        owner = self.load_owner_settings()
        hours = self.load_operating_hours()
        
        mode = owner.get("mode")
        memo = owner.get("memo")
        waktu_buka = hours.get("buka")
        waktu_tutup = hours.get("tutup")
        hari_tutup_list = hours.get("hari_tutup", [])
        
        print(f"[SitiAI] Mode dari Firebase: {mode}")
        
        if mode == "BUKA":
            return {"status":"BUKA","icon":"🟢","sebab":memo or "Dibuka khas!","memo_owner":memo}
        if mode == "TUTUP":
            return {"status":"TUTUP","icon":"🔴","sebab":memo or "Ditutup.","memo_owner":memo}
        
        if not waktu_buka or not waktu_tutup:
            return {"status":"TUTUP","icon":"🔴","sebab":"Waktu operasi belum ditetapkan."}
        
        try:
            b = waktu_buka.split(":"); t = waktu_tutup.split(":")
            buka = int(b[0])*60+int(b[1]); tutup = int(t[0])*60+int(t[1])
            if tutup == 0: tutup = 24*60
        except:
            return {"status":"TUTUP","icon":"🔴","sebab":"Ralat waktu."}
        
        masa = self.get_malaysia_time()
        hari_num = masa["hari_num"]
        now = datetime.now(timezone(timedelta(hours=8)))
        current = now.hour*60+now.minute
        hari_names = ["Ahad","Isnin","Selasa","Rabu","Khamis","Jumaat","Sabtu"]
        
        if hari_num in hari_tutup_list:
            return {"status":"TUTUP","icon":"🔴","sebab":f"Hari {hari_names[hari_num]} — tutup.","memo_owner":memo}
        if buka <= current < tutup:
            baki = tutup - current; j,m = divmod(baki,60)
            return {"status":"BUKA","icon":"🟢","sebab":f"Beroperasi. Tutup {j}j {m}m lagi.","memo_owner":memo}
        if current < buka:
            baki = buka - current; j,m = divmod(baki,60)
            return {"status":"TUTUP","icon":"🔴","sebab":f"Belum buka. Buka {j}j {m}m lagi.","memo_owner":memo}
        return {"status":"TUTUP","icon":"","sebab":"Sudah tutup.","memo_owner":memo}

    def get_system_prompt(self):
        masa = self.get_malaysia_time()
        status = self.check_kedai_status()
        menu_items = self.load_menu()
        owner = self.load_owner_settings()
        hours = self.load_operating_hours()
        
        menu_list = "\n".join([f"- **{m['nama']}** — *{m['desc']}* — `RM{m['harga']:.2f}`" for m in menu_items])
        hari_tutup_list = hours.get("hari_tutup", [])
        hari_names = ["Ahad","Isnin","Selasa","Rabu","Khamis","Jumaat","Sabtu"]
        hari_tutup_names = [hari_names[d] for d in hari_tutup_list if 0 <= d <= 6]
        
        status_icon = status.get('icon', '🟢')
        
        return f"""Anda adalah {self.persona['watak']['nama']}, {self.persona['watak']['peranan']}.
Persona: {self.persona['watak']['jantina']} Melayu {self.persona['watak']['umur']} tahun, {', '.join(self.persona['watak']['gaya'])}.

⏰ Sekarang: {masa['waktu_penuh']}
{status_icon} Status: **{status['status']}** — {status['sebab']}
📝 Memo Owner: {owner.get('memo') or 'Tiada'}

📍 {self.persona['kedai']['nama']} — {self.persona['kedai']['lokasi']}
🗺️ Maps: {self.persona['kedai']['google_maps']} | Waze: {self.persona['kedai']['waze']}
📲 WhatsApp: {self.persona['kedai']['whatsapp']}
📅 Hari Tutup: {', '.join(hari_tutup_names) if hari_tutup_names else 'Tiada'}

🍗 MENU:
{menu_list}

 Gaya: {self.persona['watak']['sapaan']} | Catchphrase: {', '.join(self.persona['watak']['catchphrase'])}
📋 Markdown: Bold **menu** | Italic *sedap* | Code `RM5` | Bullet -
✅ {chr(10).join(['- '+x for x in self.prompt['wajib']])}
🚫 {chr(10).join(['- '+x for x in self.prompt['larangan']])}"""

    # ==============================================
    # 💬 CHAT FUNCTION — DENGAN KESELAMATAN PENUH
    # ==============================================
    def chat(self, user_message, history=None):
        # 🔥 SECURITY CHECK 1: Message length limit
        MAX_INPUT = 500
        if len(user_message) > MAX_INPUT:
            return f"⚠️ *Maaf, mesej terlalu panjang!* Sila ringkaskan kepada {MAX_INPUT} aksara."
        
        #  SECURITY CHECK 2: Anti-spam (guna "anonymous" sebagai default user_id)
        user_id = "anonymous"  # Boleh tukar jika ada user authentication
        is_spam, spam_reason = self.check_spam(user_id, user_message)
        if is_spam:
            return spam_reason
        
        # 🔥 MEMORY: Use provided history or empty list
        if history is None:
            history = []
        
        #  MEMORY: Limit history to last 10 messages
        MAX_HISTORY = 10
        if len(history) > MAX_HISTORY:
            history = history[-MAX_HISTORY:]
        
        # Build conversation contents
        contents = [
            types.Content(role="user", parts=[types.Part.from_text(text=self.get_system_prompt())])
        ]
        
        for msg in history:
            contents.append(types.Content(role=msg["role"], parts=[types.Part.from_text(text=msg["text"])]))
        
        contents.append(types.Content(role="user", parts=[types.Part.from_text(text=user_message)]))
        
        config = types.GenerateContentConfig(
            top_p=0.45,
            temperature=0.7,
            thinking_config=types.ThinkingConfig(thinking_level="MINIMAL"),
        )
        
        try:
            response_text = ""
            for chunk in self.client.models.generate_content_stream(
                model=self.model, contents=contents, config=config
            ):
                if chunk.text:
                    response_text += chunk.text
            return response_text
        except Exception as e:
            print(f"[SitiAI] Error generating response: {e}")
            return "😔 Maaf, Siti AI ada masalah teknikal."

# Singleton
siti_ai = SitiAI()
