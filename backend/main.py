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
# 🗑️ PADAM GAMBAR DARI IMGBB (WITH FULL DEBUGGING)
# ==============================================
@app.route("/api/delete-image", methods=["POST"])
def delete_image():
    """Padam gambar dari ImgBB"""
    print('\n' + '='*60)
    print('🗑️ DELETE IMAGE REQUEST')
    print('='*60)
    
    try:
        data = request.json
        print(f'📥 Request data: {data}')
        
        image_id = data.get('image_id')
        print(f'📥 image_id: {image_id}')
        
        if not image_id:
            print('❌ ERROR: Tiada image_id')
            return jsonify({
                'success': False, 
                'message': 'Tiada image ID'
            }), 400
        
        imgbb_api_key = os.environ.get('IMGBB_API_KEY')
        print(f'🔑 API key exists: {bool(imgbb_api_key)}')
        
        if not imgbb_api_key:
            print('❌ ERROR: IMGBB_API_KEY tidak ditetapkan')
            return jsonify({
                'success': False, 
                'message': 'API key tidak ditetapkan di server'
            }), 500
        
        # 🔥 CUBA 3 CARA DELETE
        methods_tried = []
        
        # CARA 1: Guna endpoint delete
        try:
            print('📡 Mencuba CARA 1: POST /1/delete...')
            delete_url = 'https://api.imgbb.com/1/delete'
            response1 = requests.post(
                delete_url,
                params={
                    'key': imgbb_api_key,
                    'id': image_id
                },
                timeout=15
            )
            print(f'📥 Response 1 status: {response1.status_code}')
            print(f'📥 Response 1 text: {response1.text[:200]}')
            
            try:
                resp_data1 = response1.json()
                print(f' Response 1 JSON: {resp_data1}')
                
                if response1.status_code == 200 and resp_data1.get('success'):
                    print('✅ CARA 1 BERJAYA!')
                    return jsonify({
                        'success': True,
                        'message': 'Gambar berjaya dipadam',
                        'method': 'CARA 1'
                    })
            except:
                pass
            
            methods_tried.append(f'CARA 1: {response1.status_code}')
        except Exception as e1:
            print(f'❌ CARA 1 gagal: {e1}')
            methods_tried.append(f'CARA 1 ERROR: {str(e1)}')
        
        # CARA 2: Guna endpoint image dengan DELETE method
        try:
            print('📡 Mencuba CARA 2: DELETE /1/image/{id}...')
            image_url = f'https://api.imgbb.com/1/image/{image_id}'
            response2 = requests.delete(
                image_url,
                params={'key': imgbb_api_key},
                timeout=15
            )
            print(f'📥 Response 2 status: {response2.status_code}')
            print(f'📥 Response 2 text: {response2.text[:200]}')
            
            if response2.status_code in [200, 204, 302]:
                print('✅ CARA 2 BERJAYA!')
                return jsonify({
                    'success': True,
                    'message': 'Gambar berjaya dipadam',
                    'method': 'CARA 2'
                })
            
            methods_tried.append(f'CARA 2: {response2.status_code}')
        except Exception as e2:
            print(f' CARA 2 gagal: {e2}')
            methods_tried.append(f'CARA 2 ERROR: {str(e2)}')
        
        # CARA 3: Guna endpoint image dengan POST method
        try:
            print('📡 Mencuba CARA 3: POST /1/image/{id}...')
            image_url = f'https://api.imgbb.com/1/image/{image_id}'
            response3 = requests.post(
                image_url,
                params={
                    'key': imgbb_api_key,
                    'action': 'delete'
                },
                timeout=15
            )
            print(f'📥 Response 3 status: {response3.status_code}')
            print(f'📥 Response 3 text: {response3.text[:200]}')
            
            try:
                resp_data3 = response3.json()
                print(f' Response 3 JSON: {resp_data3}')
                
                if response3.status_code == 200:
                    print('✅ CARA 3 BERJAYA!')
                    return jsonify({
                        'success': True,
                        'message': 'Gambar berjaya dipadam',
                        'method': 'CARA 3'
                    })
            except:
                pass
            
            methods_tried.append(f'CARA 3: {response3.status_code}')
        except Exception as e3:
            print(f'❌ CARA 3 gagal: {e3}')
            methods_tried.append(f'CARA 3 ERROR: {str(e3)}')
        
        # Semua cara gagal
        print('❌ SEMUA CARA GAGAL')
        print(f'Methods tried: {methods_tried}')
        
        return jsonify({
            'success': False,
            'message': f'Gagal padam. Methods: {methods_tried}',
            'debug': methods_tried
        }), 500
            
    except Exception as e:
        print(f'❌ Unexpected error: {e}')
        import traceback
        traceback.print_exc()
        return jsonify({
            'success': False, 
            'message': f'Server error: {str(e)}'
        }), 500

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
