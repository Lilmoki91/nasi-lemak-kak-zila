import os
import json
from datetime import datetime, timezone, timedelta
from google import genai
from google.genai import types
import firebase_admin
from firebase_admin import credentials, firestore

class SitiAI:
    def init(self):
        self.client = genai.Client(
            apikey=os.environ.get("GEMINIAPI_KEY"),
        )
        self.model = "gemma-4-26b-a4b-it"
        
        # Load persona & prompt dari JSON
        with open('persona.json', 'r', encoding='utf-8') as f:
            self.persona = json.load(f)
        with open('prompt.json', 'r', encoding='utf-8') as f:
            self.prompt = json.load(f)

        # 🔥 INIT FIREBASE DARI .env
        if not firebaseadmin.apps:
            firebasecreds = json.loads(os.environ.get("FIREBASECREDENTIALS", "{}"))
            cred = credentials.Certificate(firebase_creds)
            firebaseadmin.initializeapp(cred)
        self.db = firestore.client()
        print('✅ Firebase siap!')

    # ==============================================
    # 🕐 MASA MALAYSIA (GMT+8)
    # ==============================================
    def getmalaysiatime(self):
        tz = timezone(timedelta(hours=8))
        now = datetime.now(tz)
    
        hari_map = {
            "Monday": "Isnin", "Tuesday": "Selasa", "Wednesday": "Rabu",
            "Thursday": "Khamis", "Friday": "Jumaat", "Saturday": "Sabtu", "Sunday": "Ahad"
        }
        bulan_list = ["", "Januari", "Februari", "Mac", "April", "Mei", "Jun",
                      "Julai", "Ogos", "September", "Oktober", "November", "Disember"]
        hari_english = now.strftime("%A")
        haribm = harimap[hari_english]
    
        return {
            "jam_12h": now.strftime("%I:%M %p"),
            "jam_24h": now.strftime("%H:%M"),
            "hari": hari_bm,
            "harienglish": harienglish,
            "hari_num": now.weekday(),
            "tarikhpenuh": f"{now.day} {bulanlist[now.month]} {now.year}",
            "waktupenuh": f"{haribm}, {now.day} {bulan_list[now.month]} {now.year}, {now.strftime('%I:%M %p')}"
        }

    # ==============================================
    # 📋 LOAD OWNER SETTINGS (DARI FIREBASE - TANPA FALLBACK)
    # ==============================================
    def loadownersettings(self):
        """Baca settings kedai dari Firebase — TIADA FALLBACK"""
        try:
            doc = self.db.collection("settings").document("shop_settings").get()
            if not doc.exists:
                raise Exception("Document 'shop_settings' tidak wujud dalam Firebase")
            return doc.to_dict()
        except Exception as e:
            print(f"❌ Gagal baca Firebase shop_settings: {e}")
            raise  #  THROW ERROR — TIADA FALLBACK

    # ==============================================
    # ⏰ LOAD WAKTU OPERASI (DARI FIREBASE - TANPA FALLBACK)
    # ==============================================
    def loadoperatinghours(self):
        """Baca waktu operasi dari Firebase — TIADA FALLBACK"""
        try:
            doc = self.db.collection("settings").document("operating_hours").get()
            if not doc.exists:
                raise Exception("Document 'operating_hours' tidak wujud dalam Firebase")
            return doc.to_dict()
        except Exception as e:
            print(f"❌ Gagal baca Firebase operating_hours: {e}")
            raise  # 🔥 THROW ERROR — TIADA FALLBACK

    # ==============================================
    # 🍗 LOAD MENU (DARI FIREBASE - TANPA FALLBACK)
    # ==============================================
    def load_menu(self):
        """Baca menu dari Firebase — TIADA FALLBACK"""
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
            if not menu:
                raise Exception("Tiada menu aktif dalam Firebase")
            return menu
        except Exception as e:
            print(f"❌ Gagal baca menu Firebase: {e}")
            raise  # 🔥 THROW ERROR — TIADA FALLBACK

    # ==============================================
    # 🟢 STATUS KEDAI (BACA DARI FIREBASE - TANPA FALLBACK)
    # ==============================================
    def checkkedaistatus(self):
        owner = self.loadownersettings()          # 🔥 Dari Firebase
        hours = self.loadoperatinghours()         # 🔥 Dari Firebase
        
        mode = owner.get("mode", "AUTO")
        memo = owner.get("memo", "")
        
        # 🔥 BACA WAKTU DARI FIREBASE (field name sama dengan frontend)
        waktubukastr = hours.get("buka")
        waktututupstr = hours.get("tutup")
        haritutuplist = hours.get("hari_tutup", [])
        
        if not waktubukastr or not waktututupstr:
            raise Exception("Field 'buka' atau 'tutup' tiada dalam operating_hours Firebase")
        
        # Parse waktu
        try:
            bukaparts = waktubuka_str.split(":")
            tutupparts = waktututup_str.split(":")
            buka = int(bukaparts[0]) * 60 + int(bukaparts[1])
            tutup = int(tutupparts[0]) * 60 + int(tutupparts[1])
            if tutup == 0:
                tutup = 24 * 60
        except Exception as e:
            raise Exception(f"Format waktu tidak sah: {waktubukastr} / {waktututupstr}")
        
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
        
        waktubukadisplay = format_waktu(buka)
        waktututupdisplay = format_waktu(tutup)
        
        # Nama hari dalam BM
        hari_names = ["Ahad", "Isnin", "Selasa", "Rabu", "Khamis", "Jumaat", "Sabtu"]
        
        # ✅ OVERRIDE MODE
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
        
        # ✅ AUTO MODE
        masa = self.getmalaysiatime()
        harinum = masa["harinum"]
        now = datetime.now(timezone(timedelta(hours=8)))
        current_minutes = now.hour * 60 + now.minute
        
        # Check hari tutup
        if harinum in haritutup_list:
            nexthari = (harinum + 1) % 7
            while nexthari in haritutup_list:
                nexthari = (nexthari + 1) % 7
            result = {
                "status": "TUTUP",
                "sebab": f"Hari {harinames[harinum]} — kedai tutup",
                "nextbuka": f"{harinames[nexthari]}, {waktubuka_display}"
            }
        elif currentminutes >= buka and currentminutes  MAXINPUT:
            return f"⚠️ Maaf, mesej terlalu panjang! Sila ringkaskan kepada {MAX_INPUT} aksara."
        
        if history:
            last_msgs = [msg["text"] for msg in history if msg["role"] == "user"]
            if lastmsgs and lastmsgs[-1] == user_message:
                return " Anda dah hantar mesej yang sama. Ada soalan lain?"
        
        if history is None:
            history = []
        
        contents = [
            types.Content(role="user", parts=[types.Part.fromtext(text=self.getsystem_prompt())])
        ]
        for msg in history:
            contents.append(types.Content(role=msg["role"], parts=[types.Part.from_text(text=msg["text"])]))
        contents.append(types.Content(role="user", parts=[types.Part.fromtext(text=usermessage)]))
        
        config = types.GenerateContentConfig(
            top_p=0.45, temperature=0.7,
            thinkingconfig=types.ThinkingConfig(thinkinglevel="MINIMAL"),
        )
        
        response_text = ""
        for chunk in self.client.models.generatecontentstream(
            model=self.model, contents=contents, config=config
        ):
            if chunk.text:
                response_text += chunk.text
        
        return response_text

Singleton
siti_ai = SitiAI()
