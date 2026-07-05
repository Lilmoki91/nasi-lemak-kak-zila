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
        
        with open('persona.json', 'r', encoding='utf-8') as f:
            self.persona = json.load(f)
        with open('prompt.json', 'r', encoding='utf-8') as f:
            self.prompt = json.load(f)

        if not firebase_admin._apps:
            firebase_creds = json.loads(os.environ.get("FIREBASE_CREDENTIALS", "{}"))
            cred = credentials.Certificate(firebase_creds)
            firebase_admin.initialize_app(cred)
        self.db = firestore.client()

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

    # ==============================================
    # 📋 LOAD OWNER SETTINGS — 100% FIREBASE!
    # ==============================================
    def load_owner_settings(self):
        try:
            doc = self.db.collection("settings").document("shop_settings").get()
            if doc.exists:
                return doc.to_dict()
        except:
            pass
        return {}

    # ==============================================
    # ⏰ LOAD OPERATING HOURS — 100% FIREBASE!
    # ==============================================
    def load_operating_hours(self):
        try:
            doc = self.db.collection("settings").document("operating_hours").get()
            if doc.exists:
                return doc.to_dict()
        except:
            pass
        return {}

    # ==============================================
    # 🍗 LOAD MENU — 100% FIREBASE!
    # ==============================================
    def load_menu(self):
        try:
            docs = self.db.collection("menu").where("aktif","==",True).stream()
            menu = []
            for doc in docs:
                d = doc.to_dict()
                menu.append({"nama":d.get("nama",""),"desc":d.get("desc",""),"harga":d.get("harga",0),"featured":d.get("featured",False)})
            return menu
        except:
            return []

    # ==============================================
    # 🟢 STATUS KEDAI
    # ==============================================
    def check_kedai_status(self):
        owner = self.load_owner_settings()
        hours = self.load_operating_hours()
        
        mode = owner.get("mode", None)
        memo = owner.get("memo", None)
        waktu_buka = hours.get("buka", None)
        waktu_tutup = hours.get("tutup", None)
        hari_tutup_list = hours.get("hari_tutup", [])
        
        if mode == "BUKA":
            return {"status":"BUKA","sebab":memo or "Dibuka khas!","override":True}
        if mode == "TUTUP":
            return {"status":"TUTUP","sebab":memo or "Ditutup.","override":True}
        
        if not waktu_buka or not waktu_tutup:
            return {"status":"TUTUP","sebab":"Waktu operasi belum ditetapkan."}
        
        try:
            b = waktu_buka.split(":"); t = waktu_tutup.split(":")
            buka = int(b[0])*60+int(b[1]); tutup = int(t[0])*60+int(t[1])
            if tutup == 0: tutup = 24*60
        except:
            return {"status":"TUTUP","sebab":"Ralat waktu operasi."}
        
        masa = self.get_malaysia_time()
        hari_num = masa["hari_num"]
        now = datetime.now(timezone(timedelta(hours=8)))
        current = now.hour*60+now.minute
        
        hari_names = ["Ahad","Isnin","Selasa","Rabu","Khamis","Jumaat","Sabtu"]
        
        if hari_num in hari_tutup_list:
            return {"status":"TUTUP","sebab":f"Hari {hari_names[hari_num]} — tutup."}
        if buka <= current < tutup:
            baki = tutup - current; j,m = divmod(baki,60)
            return {"status":"BUKA","sebab":f"Beroperasi. Tutup {j}j {m}m lagi."}
        if current < buka:
            baki = buka - current; j,m = divmod(baki,60)
            return {"status":"TUTUP","sebab":f"Belum buka. Buka {j}j {m}m lagi."}
        
        return {"status":"TUTUP","sebab":"Sudah tutup."}

    def get_system_prompt(self):
        masa = self.get_malaysia_time()
        status = self.check_kedai_status()
        menu_items = self.load_menu()
        
        menu_list = "\n".join([f"- **{m['nama']}** — *{m['desc']}* — `RM{m['harga']:.2f}`" for m in menu_items])
        
        return f"""Anda adalah {self.persona['watak']['nama']}, {self.persona['watak']['peranan']}.
Persona: {self.persona['watak']['jantina']} Melayu {self.persona['watak']['umur']} tahun, {', '.join(self.persona['watak']['gaya'])}.

⏰ Sekarang: {masa['waktu_penuh']}
🟢 Status: **{status['status']}** — {status['sebab']}

📍 {self.persona['kedai']['nama']} — {self.persona['kedai']['lokasi']}
🗺️ Maps: {self.persona['kedai']['google_maps']} | Waze: {self.persona['kedai']['waze']}
📲 WhatsApp: {self.persona['kedai']['whatsapp']}

🍗 MENU:
{menu_list}

🎤 Gaya: {self.persona['watak']['sapaan']} | Catchphrase: {', '.join(self.persona['watak']['catchphrase'])}

📋 Markdown: Bold **menu** | Italic *sedap* | Code `RM5` | Bullet -

✅ {chr(10).join(['- '+x for x in self.prompt['wajib']])}
🚫 {chr(10).join(['- '+x for x in self.prompt['larangan']])}"""

    def chat(self, user_message, history=None):
        if len(user_message) > 500:
            return "⚠️ Mesej terlalu panjang."
        if history and history[-1]["text"] == user_message:
            return "🤔 Mesej sama."
        if history is None: history = []
        
        contents = [types.Content(role="user", parts=[types.Part.from_text(text=self.get_system_prompt())])]
        for msg in history:
            contents.append(types.Content(role=msg["role"], parts=[types.Part.from_text(text=msg["text"])]))
        contents.append(types.Content(role="user", parts=[types.Part.from_text(text=user_message)]))
        
        config = types.GenerateContentConfig(top_p=0.45, temperature=0.7, thinking_config=types.ThinkingConfig(thinking_level="MINIMAL"))
        return "".join([chunk.text for chunk in self.client.models.generate_content_stream(model=self.model, contents=contents, config=config) if chunk.text])

siti_ai = SitiAI()
