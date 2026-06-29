# ==============================================
# 🔐 ADMIN LOGIN - ZILA FOOD
# Nombor WhatsApp + Password Authentication
# ==============================================

import hashlib
import os
import time

# Rate limiting storage
login_attempts = {}
active_tokens = {}

def hash_text(text):
    return hashlib.sha256(text.encode()).hexdigest()

def verify_admin(phone, password):
    current_time = time.time()
    
    # Rate limit check
    if phone in login_attempts:
        attempts, last_time = login_attempts[phone]
        if current_time - last_time > 3600:
            login_attempts.pop(phone, None)
        elif attempts >= 3:
            baki = int(3600 - (current_time - last_time))
            minit = baki // 60
            return {
                "success": False,
                "message": f"🔒 Terlalu banyak cubaan. Sila tunggu {minit} minit lagi.",
                "locked": True,
                "retry_after": baki
            }
    
    # Verify phone & password
    phone_hash = hash_text(phone)
    pass_hash = hash_text(password)
    
    stored_phone = os.environ.get("ADMIN_PHONE_HASH", "")
    stored_pass = os.environ.get("ADMIN_PASSWORD_HASH", "")
    
    if phone_hash == stored_phone and pass_hash == stored_pass:
        login_attempts.pop(phone, None)
        token = hash_text(f"{phone}{current_time}{os.environ.get('SECRET_KEY', '')}")
        token_expiry = current_time + (24 * 60 * 60)
        active_tokens[token] = token_expiry
        clean_expired_tokens()
        
        return {
            "success": True,
            "message": "✅ Login berjaya! Selamat datang Kak Zila.",
            "token": token,
            "expires_in": 86400
        }
    
    attempts = login_attempts.get(phone, (0, 0))[0] + 1
    login_attempts[phone] = (attempts, current_time)
    baki = 3 - attempts
    
    if baki <= 0:
        return {
            "success": False,
            "message": "🔒 Terlalu banyak cubaan. Akaun dikunci selama 1 jam.",
            "locked": True,
            "retry_after": 3600
        }
    
    return {
        "success": False,
        "message": f"❌ Nombor atau password salah. Baki cubaan: {baki}",
        "locked": False,
        "attempts_left": baki
    }

def verify_token(token):
    if token in active_tokens:
        expiry = active_tokens[token]
        if time.time() < expiry:
            return True
        else:
            active_tokens.pop(token, None)
    return False

def clean_expired_tokens():
    current_time = time.time()
    expired = [t for t, e in active_tokens.items() if current_time > e]
    for token in expired:
        active_tokens.pop(token, None)

def logout_admin(token):
    active_tokens.pop(token, None)
    return {"success": True, "message": "👋 Logout berjaya!"}

def get_login_status(phone):
    if phone in login_attempts:
        attempts, last_time = login_attempts[phone]
        current_time = time.time()
        if current_time - last_time < 3600 and attempts >= 3:
            baki = int(3600 - (current_time - last_time))
            return {
                "locked": True,
                "retry_after": baki,
                "message": f"🔒 Akaun dikunci. Cuba lagi dalam {baki // 60} minit."
            }
    return {"locked": False, "message": "✅ Sila masukkan nombor dan password."}

def reset_attempts(phone):
    login_attempts.pop(phone, None)
    return {"success": True, "message": "🔄 Attempts reset."}
