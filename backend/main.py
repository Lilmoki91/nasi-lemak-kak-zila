from flask import Flask, request, jsonify
from flask_cors import CORS
from siti_ai import siti_ai
from admin_login.auth import verify_admin, verify_token, logout_admin
import json
import os
from datetime import datetime
import firebase_admin
from firebase_admin import credentials, firestore
import requests
import base64

app = Flask(__name__)
CORS(app)  # Enable CORS untuk semua route

# ==============================================
# 🔥 FIREBASE INIT
# ==============================================
if not firebase_admin._apps:
    firebase_creds = json.loads(os.environ.get("FIREBASE_CREDENTIALS", "{}"))
    cred = credentials.Certificate(firebase_creds)
    firebase_admin.initialize_app(cred)
db = firestore.client()

conversations = {}

# ==============================================
# 💬 CHAT API
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
    return jsonify({"response": response, "session_id": session_id})

# ==============================================
# 🍗 MENU API
# ==============================================
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
    except Exception as e:
        print(f'❌ Error get_menu: {e}')
        return jsonify([])

# ==============================================
# ℹ️ INFO API
# ==============================================
@app.route("/api/info", methods=["GET"])
def get_info():
    return jsonify({
        "nama": "Zila Food",
        "lokasi": "PPR Sri Pantai Blok 102",
        "whatsapp": "011-1164 0776"
    })

# ==============================================
# 🔐 ADMIN AUTH API
# ==============================================
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

# ==============================================
# ⚙️ ADMIN SETTINGS API
# ==============================================
@app.route("/api/admin/settings", methods=["GET"])
def get_owner_settings():
    try:
        doc = db.collection("settings").document("shop_settings").get()
        if doc.exists:
            return jsonify(doc.to_dict())
    except Exception as e:
        print(f'❌ Error get_owner_settings: {e}')
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

# ==============================================
# 📸 UPLOAD GAMBAR KE IMGBB (SIMPAN ID UNTUK DELETE)
# ==============================================
@app.route("/api/upload-image", methods=["POST"])
def upload_image():
    """Upload gambar & simpan image_id untuk delete nanti"""
    try:
        if 'image' not in request.files:
            return jsonify({'success': False, 'message': 'Tiada fail gambar'}), 400
        
        file = request.files['image']
        if file.filename == '':
            return jsonify({'success': False, 'message': 'Tiada fail dipilih'}), 400
        
        allowed_types = {'image/jpeg', 'image/png', 'image/webp'}
        if file.content_type not in allowed_types:
            return jsonify({
                'success': False, 
                'message': f'Format tidak sah: {file.content_type}'
            }), 400
        
        imgbb_api_key = os.environ.get('IMGBB_API_KEY')
        if not imgbb_api_key:
            return jsonify({'success': False, 'message': 'API key tidak ditetapkan'}), 500
        
        file.seek(0, 2)
        file_size = file.tell()
        file.seek(0)
        
        if file_size > 32 * 1024 * 1024:
            return jsonify({'success': False, 'message': 'Fail terlalu besar'}), 400
        
        print(f'📤 Upload: {file.filename} ({file_size / 1024:.2f}KB)')
        
        file_bytes = file.read()
        file_base64 = base64.b64encode(file_bytes).decode('utf-8')
        
        response = requests.post(
            'https://api.imgbb.com/1/upload',
            data={
                'key': imgbb_api_key,
                'image': file_base64,
                'name': f'menu_{int(datetime.now().timestamp())}'
            },
            timeout=30
        )
        
        if response.status_code != 200:
            print(f'❌ Upload gagal: {response.text}')
            return jsonify({'success': False, 'message': 'Upload gagal'}), 500
        
        data = response.json()
        
        if data.get('success'):
            image_url = data['data']['display_url']
            image_id = data['data']['id']  # 🔥 PENTING: Simpan ID untuk delete
            
            print(f'✅ Upload berjaya: {image_url}')
            print(f'🆔 Image ID: {image_id}')
            
            return jsonify({
                'success': True,
                'url': image_url,
                'image_id': image_id,  # 🔥 Return ID
                'size': file_size
            })
        else:
            return jsonify({'success': False, 'message': 'Upload gagal'}), 500
            
    except Exception as e:
        print(f'❌ Error: {e}')
        import traceback
        traceback.print_exc()
        return jsonify({'success': False, 'message': str(e)}), 500


# ==============================================
# 🗑️ PADAM GAMBAR DARI IMGBB (GUNA ID + API KEY)
# ==============================================
@app.route("/api/delete-image", methods=["POST"])
def delete_image():
    """Padam gambar guna image_id + API key (cara yang BETUL)"""
    try:
        data = request.json
        image_id = data.get('image_id')
        
        print(f'\n{"="*60}')
        print(f'🗑️ DELETE IMAGE REQUEST')
        print(f'{"="*60}')
        print(f'📥 Received image_id: {image_id}')
        
        if not image_id:
            print('❌ Tiada image_id')
            return jsonify({'success': False, 'message': 'Tiada image ID'}), 400
        
        imgbb_api_key = os.environ.get('IMGBB_API_KEY')
        if not imgbb_api_key:
            return jsonify({'success': False, 'message': 'API key tidak ditetapkan'}), 500
        
        # 🔥 GUNA API DELETE YANG BETUL
        delete_api_url = f'https://api.imgbb.com/1/delete?key={imgbb_api_key}&id={image_id}'
        
        print(f'📡 Calling ImgBB DELETE API...')
        response = requests.post(delete_api_url, timeout=15)
        
        print(f'📥 ImgBB response status: {response.status_code}')
        
        try:
            response_data = response.json()
            print(f'📥 Response: {response_data}')
            
            if response_data.get('success'):
                print('✅ Gambar BERJAYA dipadam!')
                return jsonify({
                    'success': True,
                    'message': 'Gambar berjaya dipadam'
                })
            else:
                error_msg = response_data.get('error', {}).get('message', 'Unknown error')
                print(f'⚠️ Gagal padam: {error_msg}')
                return jsonify({
                    'success': False,
                    'message': error_msg
                }), 500
        except:
            # Kalau response bukan JSON
            if response.status_code in [200, 302]:
                return jsonify({'success': True, 'message': 'Gambar dipadam'})
            return jsonify({'success': False, 'message': 'Gagal padam'}), 500
            
    except requests.exceptions.Timeout:
        return jsonify({'success': False, 'message': 'Delete timeout'}), 504
    except Exception as e:
        print(f'❌ Delete error: {e}')
        import traceback
        traceback.print_exc()
        return jsonify({'success': False, 'message': str(e)}), 500

# ==============================================
# 🏠 HOME
# ==============================================
@app.route("/")
def home():
    return jsonify({
        "name": "Zila Food API",
        "status": "running",
        "version": "2.0",
        "features": ["chat", "menu", "admin", "image-upload", "image-delete"]
    })


# ==============================================
# 🚀 RUN APP
# ==============================================
if __name__ == "__main__":
    port = int(os.environ.get("PORT", 8080))
    print(f'🚀 Starting server on port {port}')
    app.run(host="0.0.0.0", port=port, debug=False)
