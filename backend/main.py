from flask import Flask, request, jsonify
from flask_cors import CORS
from siti_ai import siti_ai
from admin_login.auth import verify_admin, verify_token, logout_admin, get_login_status
import json
from datetime import datetime

app = Flask(__name__)
CORS(app)

conversations = {}

# ==============================================
# 💬 API CHAT
# ==============================================
@app.route("/api/chat", methods=["POST"])
def chat():
    data = request.json
    user_message = data.get("message", "")
    session_id = data.get("session_id", "default")
    
    history = conversations.get(session_id, [])
    response = siti_ai.chat(user_message, history)
    
    history.append({"role": "user", "text": user_message})
    history.append({"role": "model", "text": response})
    conversations[session_id] = history[-10:]
    
    return jsonify({
        "response": response,
        "session_id": session_id
    })

# ==============================================
# 🍗 API MENU
# ==============================================
@app.route("/api/menu", methods=["GET"])
def get_menu():
    menu = [
        {"id": 1, "nama": "Nasi Lemak Berlauk", "desc": "Ayam goreng, telur, sambal, ikan bilis", "harga": 5.00},
        {"id": 2, "nama": "Nasi Lemak Biasa", "desc": "Nasi lemak klasik dengan sambal", "harga": 2.00},
        {"id": 3, "nama": "Kaaripuf", "desc": "Karipap rangup & sedap", "harga": 1.00},
        {"id": 4, "nama": "Air Balang", "desc": "Minuman segar menyegarkan", "harga": 1.00},
    ]
    return jsonify(menu)

# ==============================================
# 📋 API INFO
# ==============================================
@app.route("/api/info", methods=["GET"])
def get_info():
    info = {
        "nama": "Zila Food (Nasi Lemak Kak Zila)",
        "lokasi": "PPR Sri Pantai Blok 102, Kuala Lumpur",
        "waktu": "7:30PM - 12:00AM",
        "hari": "Jumaat - Rabu",
        "tutup": "Khamis",
        "whatsapp": "011-1164 0776"
    }
    return jsonify(info)

# ==============================================
# 🔐 API ADMIN LOGIN
# ==============================================
@app.route("/api/admin/login", methods=["POST"])
def admin_login():
    data = request.json
    phone = data.get("phone", "").strip()
    password = data.get("password", "").strip()
    
    if not phone or not password:
        return jsonify({"success": False, "message": "❌ Sila isi nombor dan password."})
    
    result = verify_admin(phone, password)
    return jsonify(result)

# ==============================================
# 🔍 API VERIFY TOKEN
# ==============================================
@app.route("/api/admin/verify", methods=["POST"])
def admin_verify():
    data = request.json
    token = data.get("token", "")
    valid = verify_token(token)
    return jsonify({"valid": valid})

# ==============================================
# 📊 API LOGIN STATUS
# ==============================================
@app.route("/api/admin/status", methods=["POST"])
def admin_status():
    data = request.json
    phone = data.get("phone", "").strip()
    result = get_login_status(phone)
    return jsonify(result)

# ==============================================
# 🚪 API LOGOUT
# ==============================================
@app.route("/api/admin/logout", methods=["POST"])
def admin_logout():
    data = request.json
    token = data.get("token", "")
    result = logout_admin(token)
    return jsonify(result)

# ==============================================
# ⚙️ API OWNER SETTINGS (GET)
# ==============================================
@app.route("/api/admin/settings", methods=["GET"])
def get_owner_settings():
    try:
        with open('owner_settings.json', 'r') as f:
            settings = json.load(f)
    except:
        settings = {"mode": "AUTO", "memo": "", "last_updated": ""}
    return jsonify(settings)

# ==============================================
# ⚙️ API OWNER SETTINGS (UPDATE)
# ==============================================
@app.route("/api/admin/settings", methods=["POST"])
def update_owner_settings():
    data = request.json
    token = data.get("token", "")
    
    if not verify_token(token):
        return jsonify({"success": False, "message": "❌ Sesi tamat. Sila login semula."}), 401
    
    settings = {
        "mode": data.get("mode", "AUTO"),
        "memo": data.get("memo", ""),
        "last_updated": datetime.now().isoformat(),
        "updated_by": "Kak Zila"
    }
    
    with open('owner_settings.json', 'w') as f:
        json.dump(settings, f, indent=2)
    
    return jsonify({"success": True, "message": "✅ Tetapan disimpan!"})

# ==============================================
# 🏠 HOME
# ==============================================
@app.route("/")
def home():
    return jsonify({
        "name": "Zila Food API",
        "version": "1.0.0",
        "status": "running"
    })

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
