from flask import Flask, request, jsonify
from flask_cors import CORS
from siti_ai import siti_ai
import os

app = Flask(__name__)
CORS(app)

conversations = {}

# ==============================================
# 💬 CHAT API
# ==============================================
@app.route("/api/chat", methods=["POST"])
def chat():
    try:
        data = request.json
        user_message = data.get("message", "")
        session_id = data.get("session_id", "default")
        history = conversations.get(session_id, [])
        
        print(f"[AI] Processing: {user_message[:50]}")
        
        response = siti_ai.chat(user_message, history)
        
        history.append({"role": "user", "text": user_message})
        history.append({"role": "model", "text": response})
        conversations[session_id] = history[-10:]
        
        print(f"[AI] Response sent: {len(response)} chars")
        
        return jsonify({"response": response, "session_id": session_id})
    except Exception as e:
        print(f"[AI] Error: {e}")
        import traceback
        traceback.print_exc()
        return jsonify({"response": "😔 Maaf, Siti AI ada masalah teknikal.", "session_id": session_id}), 500

# ==============================================
# 🏠 HOME
# ==============================================
@app.route("/")
def home():
    return jsonify({
        "name": "Zila Food AI - Siti Chatbot",
        "status": "running",
        "version": "2.1",
        "features": ["chat"]
    })

# ==============================================
# 🚀 RUN APP
# ==============================================
if __name__ == "__main__":
    port = int(os.environ.get("PORT", 8080))
    print(f'🤖 Starting AI Chatbot on port {port}')
    app.run(host="0.0.0.0", port=port, debug=False)
