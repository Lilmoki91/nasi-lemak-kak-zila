from flask import Flask, request, jsonify
from flask_cors import CORS
from admin_login.auth import verify_admin, verify_token, logout_admin
import json
import os
from datetime import datetime
import firebase_admin
from firebase_admin import credentials, firestore
import cloudinary
import cloudinary.uploader
import cloudinary.api

app = Flask(__name__)
CORS(app)

# ==============================================
# 🔥 FIREBASE INIT
# ==============================================
if not firebase_admin._apps:
    firebase_creds = json.loads(os.environ.get("FIREBASE_CREDENTIALS", "{}"))
    cred = credentials.Certificate(firebase_creds)
    firebase_admin.initialize_app(cred)
db = firestore.client()

# ==============================================
# ☁️ CLOUDINARY INIT
# ==============================================
cloudinary.config(
    cloud_name=os.environ.get('CLOUDINARY_CLOUD_NAME'),
    api_key=os.environ.get('CLOUDINARY_API_KEY'),
    api_secret=os.environ.get('CLOUDINARY_API_SECRET')
)

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
    except:
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
#  ADMIN AUTH API
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

# ==============================================
# 📸 UPLOAD GAMBAR KE CLOUDINARY
# ==============================================
@app.route("/api/upload-image", methods=["POST"])
def upload_image():
    """Upload gambar ke Cloudinary"""
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
        
        print(f'📤 Upload ke Cloudinary: {file.filename}')
        
        # Upload ke Cloudinary
        result = cloudinary.uploader.upload(
            file,
            folder='zila-food/menu',
            resource_type='image',
            overwrite=True
        )
        
        print(f'✅ Upload berjaya: {result["secure_url"]}')
        print(f'🆔 Public ID: {result["public_id"]}')
        
        return jsonify({
            'success': True,
            'url': result['secure_url'],
            'public_id': result['public_id'],
            'size': result['bytes']
        })
        
    except Exception as e:
        print(f'❌ Upload error: {e}')
        import traceback
        traceback.print_exc()
        return jsonify({'success': False, 'message': str(e)}), 500

# ==============================================
# 🗑️ PADAM GAMBAR DARI CLOUDINARY
# ==============================================
@app.route("/api/delete-image", methods=["POST"])
def delete_image():
    """Padam gambar dari Cloudinary guna public_id"""
    try:
        data = request.json
        public_id = data.get('public_id')
        
        print(f'\n{"="*60}')
        print(f'🗑️ DELETE IMAGE REQUEST')
        print(f'{"="*60}')
        print(f'📥 Received public_id: {public_id}')
        
        if not public_id:
            print('❌ Tiada public_id')
            return jsonify({'success': False, 'message': 'Tiada public ID'}), 400
        
        # Delete dari Cloudinary
        result = cloudinary.uploader.destroy(public_id)
        
        print(f'📥 Cloudinary response: {result}')
        
        if result.get('result') == 'ok':
            print('✅ Gambar BERJAYA dipadam!')
            return jsonify({
                'success': True,
                'message': 'Gambar berjaya dipadam'
            })
        else:
            print(f'⚠️ Gagal padam: {result}')
            return jsonify({
                'success': False,
                'message': f'Gagal padam: {result.get("result", "unknown")}'
            }), 500
            
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
        "name": "Zila Food API - Main System",
        "status": "running",
        "version": "2.1",
        "features": ["menu", "admin", "image-upload", "image-delete"]
    })

# ==============================================
# 🚀 RUN APP
# ==============================================
if __name__ == "__main__":
    port = int(os.environ.get("PORT", 8080))
    print(f'🚀 Starting Main System on port {port}')
    app.run(host="0.0.0.0", port=port, debug=False)
