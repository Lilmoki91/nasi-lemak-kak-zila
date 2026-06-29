from flask import Flask, request, jsonify
from flask_cors import CORS
from siti_ai import siti_ai
from admin_login.auth import verify_admin, verify_token, logout_admin, get_login_status
import json
from datetime import datetime

app = Flask(__name__)
CORS(app)  # Allow frontend access

# Store conversation history (dalam memory — untuk production guna database)
conversations = {}

@app.route("/api/chat", methods=["POST"])
def chat():
    data = request.json
    user_message = data.get("message", "")
    session_id = data.get("session_id", "default")
    
    # Get history
    history = conversations.get(session_id, [])
    
    # Get AI response
    response = siti_ai.chat(user_message, history)
    
    # Save to history
    history.append({"role": "user", "text": user_message})
    history.append({"role": "model", "text": response})
    conversations[session_id] = history[-10:]  # Keep last 10 messages
    
    return jsonify({
        "response": response,
        "session_id": session_id
    })

@app.route("/api/menu", methods=["GET"])
def get_menu():
    menu = [
        {"id": 1, "nama": "Nasi Lemak Berlauk", "desc": "Ayam goreng, telur, sambal, ikan bilis", "harga": 5.00},
        {"id": 2, "nama": "Nasi Lemak Biasa", "desc": "Nasi lemak klasik dengan sambal", "harga": 2.00},
        {"id": 3, "nama": "Kaaripuf", "desc": "Karipap rangup & sedap", "harga": 1.00},
        {"id": 4, "nama": "Air Balang", "desc": "Minuman segar menyegarkan", "harga": 1.00},
    ]
    return jsonify(menu)

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

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
