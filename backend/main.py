from flask import Flask, request, jsonify
from flask_cors import CORS
from siti_ai import siti_ai
from admin_login.auth import verify_admin, verify_token, logout_admin
import json
import os
from datetime import datetime
import firebase_admin
from firebase_admin import credentials, firestore

app = Flask(__name__)
CORS(app)

if not firebase_admin._apps:
    firebase_creds = json.loads(os.environ.get("FIREBASE_CREDENTIALS", "{}"))
    cred = credentials.Certificate(firebase_creds)
    firebase_admin.initialize_app(cred)
db = firestore.client()

conversations = {}

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
    return jsonify({"response": response, "session_id": session_id})

@app.route("/api/menu", methods=["GET"])
def get_menu():
    try:
        docs = db.collection("menu").where("aktif", "==", True).stream()
        menu = []
        for doc in docs:
            d = doc.to_dict()
            menu.append({
                "id": doc.id,
                "nama": d.get("nama", ""),
                "desc": d.get("desc", ""),
                "harga": d.get("harga", 0),
                "gambar": d.get("gambar", "")
            })
        return jsonify(menu)
    except:
        return jsonify([])

@app.route("/api/info", methods=["GET"])
def get_info():
    return jsonify({
        "nama": "Zila Food",
        "lokasi": "PPR Sri Pantai Blok 102",
        "whatsapp": "011-1164 0776"
    })

@app.route("/api/admin/login", methods=["POST"])
def admin_login():
    data = request.json
    return jsonify(verify_admin(data.get("phone", "").strip(), data.get("password", "").strip()))

@app.route("/api/admin/verify", methods=["POST"])
def admin_verify():
    return jsonify({"valid": verify_token(request.json.get("token", ""))})

@app.route("/api/admin/logout", methods=["POST"])
def admin_logout():
    return jsonify(logout_admin(request.json.get("token", "")))

@app.route("/api/admin/settings", methods=["GET"])
def get_owner_settings():
    try:
        doc = db.collection("settings").document("shop_settings").get()
        if doc.exists:
            return jsonify(doc.to_dict())
    except:
        pass
    return jsonify({})

@app.route("/api/admin/settings", methods=["POST"])
def update_owner_settings():
    data = request.json
    if not verify_token(data.get("token", "")):
        return jsonify({"success": False, "message": "Sesi tamat"}), 401
    db.collection("settings").document("shop_settings").set({
        "mode": data.get("mode", "AUTO"),
        "memo": data.get("memo", ""),
        "last_updated": datetime.now().isoformat()
    }, merge=True)
    return jsonify({"success": True, "message": "Disimpan"})

@app.route("/")
def home():
    return jsonify({"name": "Zila Food API", "status": "running"})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
