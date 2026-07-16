// Created by Muhammad Johan

// ==============================================
// 🍽️ DATA GLOBAL
// ==============================================
let menuData = [];
let troli = {};
let ownerMode = null;
let operatingHours = { buka: null, tutup: null, hari_tutup: null };
let adminToken = null;
let currentMemo = null;
let isSitiReplying = false;
let sessionId = 'user-' + Date.now();

// 📸 Image upload state
let selectedFile = null;
let finalImageUrl = '';
let isUploading = false;
let isImageDeleted = false;

const NAMA_HARI = ['Ahad','Isnin','Selasa','Rabu','Khamis','Jumaat','Sabtu'];
const MAX_FILE_SIZE = 5 * 1024 * 1024;
const COMPRESS_MAX_WIDTH = 800;
const COMPRESS_QUALITY = 0.82;

// ==============================================
// 🔥 FIREBASE HELPERS
// ==============================================
function waitForFirebase() {
    return new Promise((resolve) => {
        if (window.firebaseDB) { resolve(); return; }
        window.addEventListener('firebase-ready', () => resolve(), { once: true });
        setTimeout(() => resolve(), 5000);
    });
}

// ==============================================
// 📸 IMAGE COMPRESSION & UPLOAD
// ==============================================
function compressImage(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = function(e) {
            const img = new Image();
            img.onload = function() {
                const canvas = document.createElement('canvas');
                let width = img.width;
                let height = img.height;
                if (width > COMPRESS_MAX_WIDTH) {
                    height = Math.round((height * COMPRESS_MAX_WIDTH) / width);
                    width = COMPRESS_MAX_WIDTH;
                }
                canvas.width = width;
                canvas.height = height;
                const ctx = canvas.getContext('2d');
                ctx.drawImage(img, 0, 0, width, height);
                canvas.toBlob(
                    (blob) => {
                        if (blob) resolve(blob);
                        else reject(new Error('Gagal compress gambar'));
                    },
                    'image/jpeg',
                    COMPRESS_QUALITY
                );
            };
            img.onerror = () => reject(new Error('Gagal baca gambar'));
            img.src = e.target.result;
        };
        reader.onerror = () => reject(new Error('Gagal baca fail'));
        reader.readAsDataURL(file);
    });
}

// ==============================================
// 📸 UPLOAD GAMBAR KE CLOUDINARY
// ==============================================
async function uploadImageToStorage(file, menuId) {
    showUploadProgress(15, 'Memampatkan gambar...');
    const compressedBlob = await compressImage(file);
    
    showUploadProgress(30, 'Memuat naik ke server...');
    
    const compressedFile = new File([compressedBlob], `menu_${menuId || Date.now()}.jpg`, { 
        type: 'image/jpeg' 
    });
    
    const formData = new FormData();
    formData.append('image', compressedFile);
    
    try {
        const response = await fetch('https://zila-food.onrender.com/api/upload-image', {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.message || 'Upload gagal');
        }
        
        const data = await response.json();
        
        if (data.success) {
            showUploadProgress(90, 'Berjaya!');
            console.log('✅ Upload data:', data);
            return {
                url: data.url,
                path: 'cloudinary:' + data.public_id,
                size: data.size
            };
        } else {
            throw new Error(data.message || 'Upload gagal');
        }
    } catch(err) {
        console.error('Upload error:', err);
        throw err;
    }
}

// ==============================================
// 🗑️ PADAM GAMBAR DARI CLOUDINARY
// ==============================================
async function deleteImageFromStorage(imagePath) {
    console.log('🗑️ deleteImageFromStorage dipanggil');
    console.log('📥 imagePath:', imagePath);
    
    if (!imagePath) {
        console.log('⚠️ imagePath kosong, skip padam');
        return;
    }
    
    if (imagePath.startsWith('cloudinary:')) {
        try {
            const publicId = imagePath.replace('cloudinary:', '');
            console.log('📤 Calling backend delete API');
            console.log('📤 public_id:', publicId);
            
            const response = await fetch('https://zila-food.onrender.com/api/delete-image', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ public_id: publicId })
            });
            
            console.log('📥 Backend response status:', response.status);
            
            const responseData = await response.json();
            console.log('📥 Backend response data:', responseData);
            
            if (response.ok && responseData.success) {
                console.log('✅ Gambar BERJAYA dipadam dari Cloudinary!');
            } else {
                console.log('⚠️ Gagal padam gambar:', responseData.message);
            }
        } catch(e) {
            console.error('❌ Error dalam deleteImageFromStorage:', e);
        }
        return;
    }
    
    console.log('⏭️ Bukan dari Cloudinary, tidak dipadam:', imagePath);
}

// ==============================================
// 📸 UI FUNCTIONS FOR IMAGE UPLOAD
// ==============================================
function handleFileSelect(event) {
    const file = event.target.files[0];
    if (!file) return;
    const validTypes = ['image/jpeg', 'image/png', 'image/webp'];
    if (!validTypes.includes(file.type)) {
        toast('❌ Format tidak sah! Guna JPG, PNG atau WEBP.');
        event.target.value = '';
        return;
    }
    if (file.size > MAX_FILE_SIZE) {
        toast('❌ Gambar terlalu besar! Max 5MB.');
        event.target.value = '';
        return;
    }
    selectedFile = file;
    finalImageUrl = '';
    document.getElementById('menuGambarUrl').value = '';
    const reader = new FileReader();
    reader.onload = function(e) {
        const previewImg = document.getElementById('uploadPreviewImg');
        const previewWrap = document.getElementById('uploadPreviewWrap');
        const placeholder = document.getElementById('uploadPlaceholder');
        const uploadArea = document.getElementById('uploadArea');
        previewImg.src = e.target.result;
        previewWrap.classList.add('show');
        placeholder.style.display = 'none';
        uploadArea.classList.add('has-image');
        document.getElementById('uploadFileName').textContent = file.name;
        document.getElementById('uploadFileSize').textContent = formatFileSize(file.size);
    };
    reader.readAsDataURL(file);
}

function removeUploadPreview(event) {
    if (event) { 
        event.stopPropagation(); 
        event.preventDefault(); 
        event.stopImmediatePropagation();
    }
    
    selectedFile = null;
    finalImageUrl = '';
    isImageDeleted = true;
    
    const previewWrap = document.getElementById('uploadPreviewWrap');
    const placeholder = document.getElementById('uploadPlaceholder');
    const uploadArea = document.getElementById('uploadArea');
    const fileInput = document.getElementById('menuGambarFile');
    const urlInput = document.getElementById('menuGambarUrl');
    
    if (previewWrap) previewWrap.classList.remove('show');
    if (placeholder) placeholder.style.display = 'block';
    if (uploadArea) uploadArea.classList.remove('has-image');
    if (fileInput) {
        fileInput.value = '';
        fileInput.type = 'text';
        fileInput.type = 'file';
    }
    if (urlInput) urlInput.value = '';
    
    console.log('✅ Gambar dibuang dari preview');
}

function loadUrlImage() {
    const url = document.getElementById('menuGambarUrl').value.trim();
    if (!url) { toast('❌ Sila masukkan URL gambar'); return; }
    try { new URL(url); } catch(e) { toast('❌ URL tidak sah'); return; }
    selectedFile = null;
    document.getElementById('menuGambarFile').value = '';
    const testImg = new Image();
    testImg.onload = function() {
        finalImageUrl = url;
        const previewImg = document.getElementById('uploadPreviewImg');
        const previewWrap = document.getElementById('uploadPreviewWrap');
        const placeholder = document.getElementById('uploadPlaceholder');
        const uploadArea = document.getElementById('uploadArea');
        previewImg.src = url;
        previewWrap.classList.add('show');
        placeholder.style.display = 'none';
        uploadArea.classList.add('has-image');
        document.getElementById('uploadFileName').textContent = 'URL luar';
        document.getElementById('uploadFileSize').textContent = '-';
        toast('✅ Gambar dari URL dimuatkan');
    };
    testImg.onerror = function() { toast('❌ Gagal load gambar dari URL ini'); };
    testImg.src = url;
}

function formatFileSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

function showUploadProgress(percent, text) {
    const progress = document.getElementById('uploadProgress');
    const fill = document.getElementById('uploadProgressFill');
    const progressText = document.getElementById('uploadProgressText');
    progress.classList.add('show');
    fill.style.width = percent + '%';
    progressText.textContent = text || `Memuat naik... ${Math.round(percent)}%`;
}

function hideUploadProgress() {
    const progress = document.getElementById('uploadProgress');
    progress.classList.remove('show');
    document.getElementById('uploadProgressFill').style.width = '0%';
}

// ==============================================
// 📋 FETCH SHOP SETTINGS DARI FIREBASE (mode + memo)
// ==============================================
async function fetchShopSettings() {
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        const docRef = window.fbDoc(db, 'settings', 'shop_settings');
        const snap = await window.fbGetDoc(docRef);
        if (snap.exists()) {
            const d = snap.data();
            ownerMode = d.mode;
            currentMemo = d.memo;
            return d;
        }
    } catch(e) {
        console.log('Gagal fetch shop settings');
    }
    return null;
}

// ==============================================
// 📢 TAMBAH: TICKER UPDATE (TAK UBAH FUNGSI LAIN)
// ==============================================
async function updateTicker() {
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        const docRef = window.fbDoc(db, 'settings', 'shop_settings');
        const snap = await window.fbGetDoc(docRef);
        if (snap.exists()) {
            const d = snap.data();
            const tickerData = d.ticker || '';
            
            const tickerContainer = document.getElementById('tickerContainer');
            const tickerTextEl = document.getElementById('tickerText');
            
            if (tickerData && tickerData.trim() !== '') {
                tickerTextEl.textContent = tickerData;
                tickerContainer.style.display = 'block';
            } else {
                tickerContainer.style.display = 'none';
            }
        }
    } catch(e) {
        console.log('Gagal load ticker');
    }
}

// ==============================================
// ⏰ WAKTU OPERASI - FIREBASE (settings/operating_hours)
// ==============================================
async function fetchWaktuOperasi() {
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        const docRef = window.fbDoc(db, 'settings', 'operating_hours');
        const snap = await window.fbGetDoc(docRef);
        if (snap.exists()) {
            const d = snap.data();
            operatingHours = {
                buka: d.buka,
                tutup: d.tutup,
                hari_tutup: d.hari_tutup
            };
        }
    } catch(e) {
        console.log('Gagal fetch waktu operasi');
    }
    updateWaktuDisplay();
}

// 🔥 FUNGSI updateWaktuDisplay() YANG BETUL - SEMBUNYI MASA & HARI BILA BUKA/TUTUP
function updateWaktuDisplay() {
    if (!operatingHours.buka || !operatingHours.tutup) return;
    
    const bukaFormatted = formatTime12(operatingHours.buka);
    const tutupFormatted = formatTime12(operatingHours.tutup);
    
    const headerWaktu = document.getElementById('headerWaktu');
    if (headerWaktu) {
        // 🔥 HANYA TUNJUK MASA OPERASI BILA MODE AUTO
        if (ownerMode === 'AUTO') {
            headerWaktu.innerHTML = `<i class="fas fa-clock" style="color:green; font-size:15px; margin-top:4px;"></i> ${bukaFormatted} - ${tutupFormatted}`;
            headerWaktu.style.display = 'block';
        } else {
            // Sembunyi bila mode BUKA/TUTUP
            headerWaktu.style.display = 'none';
        }
    }
    
    const headerHari = document.getElementById('headerHariTutup');
    if (headerHari) {
        // 🔥 HANYA TUNJUK HARI TUTUP BILA MODE AUTO
        if (ownerMode === 'AUTO' && operatingHours.hari_tutup && operatingHours.hari_tutup.length > 0) {
            const hariNames = operatingHours.hari_tutup.map(d => NAMA_HARI[d]).join(', ');
            headerHari.innerHTML = `<i class="fas fa-times" style="color:red; font-size:15px;"></i> Tutup: ${hariNames}`;
            headerHari.style.display = 'block';
        } else {
            // Sembunyi bila mode BUKA/TUTUP atau tiada hari tutup
            headerHari.style.display = 'none';
        }
    }
    
    // 🔥 LOKASI PAGE - TUNJUK JADUAL HANYA BILA MODE AUTO
    const lokasiWaktu = document.getElementById('lokasiWaktuOperasi');
    if (lokasiWaktu) {
        if (ownerMode === 'AUTO') {
            let html = '';
            for (let i = 0; i < 7; i++) {
                const isTutup = operatingHours.hari_tutup && operatingHours.hari_tutup.includes(i);
                html += `<div class="waktu-row">
                    <span class="waktu-hari">${NAMA_HARI[i]}</span>
                    <span class="waktu-masa" style="color:${isTutup ? '#EF4444' : 'var(--primary)'};">${isTutup ? '❌ Tutup' : bukaFormatted + ' - ' + tutupFormatted}</span>
                </div>`;
            }
            lokasiWaktu.innerHTML = html;
            lokasiWaktu.style.display = 'block';
        } else {
            // Sembunyi jadual lokasi bila mode BUKA/TUTUP
            lokasiWaktu.style.display = 'none';
        }
    }
}

function formatTime12(timeStr) {
    if (!timeStr) return '--:--';
    const [h, m] = timeStr.split(':').map(Number);
    const period = h >= 12 ? 'PM' : 'AM';
    const hour12 = h === 0 ? 12 : h > 12 ? h - 12 : h;
    return `${hour12}:${String(m).padStart(2,'0')} ${period}`;
}

function isHariTutup() {
    if (!operatingHours.hari_tutup) return false;
    return operatingHours.hari_tutup.includes(new Date().getDay());
}

function isWaktuOperasi() {
    if (!operatingHours.buka || !operatingHours.tutup) return false;
    const now = new Date();
    const currentMinutes = now.getHours() * 60 + now.getMinutes();
    const [bukaH, bukaM] = operatingHours.buka.split(':').map(Number);
    const [tutupH, tutupM] = operatingHours.tutup.split(':').map(Number);
    let bukaMinutes = bukaH * 60 + bukaM;
    let tutupMinutes = tutupH * 60 + tutupM;
    if (tutupMinutes <= bukaMinutes) {
        tutupMinutes += 24 * 60;
        let checkTime = currentMinutes;
        if (checkTime < bukaMinutes) checkTime += 24 * 60;
        return checkTime >= bukaMinutes && checkTime < tutupMinutes;
    }
    return currentMinutes >= bukaMinutes && currentMinutes < tutupMinutes;
}

function kedaiBuka() {
    if (!ownerMode) return false;
    if (ownerMode === 'BUKA') return true;
    if (ownerMode === 'TUTUP') return false;
    if (isHariTutup()) return false;
    if (!isWaktuOperasi()) return false;
    return true;
}

function mesejTutup() {
    if (!ownerMode) return '⏳ Memuat data...';
    if (ownerMode === 'TUTUP') return '🔴 Kedai Ditutup Oleh Owner';
    if (isHariTutup()) return `🔴 Kedai Tutup Hari ${NAMA_HARI[new Date().getDay()]}`;
    if (!isWaktuOperasi() && operatingHours.buka) {
        return `🕐 Buka: ${formatTime12(operatingHours.buka)} - ${formatTime12(operatingHours.tutup)}`;
    }
    return '';
}

// ==============================================
// 🍽️ FETCH MENU DARI FIREBASE ❌ TIADA FALLBACK
// ==============================================
async function fetchMenu() {
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        const menuRef = window.fbCollection(db, 'menu');
        const snapshot = await window.fbGetDocs(menuRef);
        menuData = [];
        snapshot.forEach(docSnap => {
            const d = docSnap.data();
            if (d.aktif !== false) {
                menuData.push({
                    id: docSnap.id,
                    nama: d.nama,
                    desc: d.desc,
                    harga: d.harga,
                    gambar: d.gambar || '',
                    gambarPath: d.gambarPath || '',
                    featured: d.featured || false
                });
            }
        });
        menuData.sort((a, b) => (b.featured ? 1 : 0) - (a.featured ? 1 : 0));
    } catch(e) {
        console.log('Gagal fetch menu dari Firebase');
    }
}

// ==============================================
// 🛒 TROLI
// ==============================================
function tambahItem(id) {
    if (!kedaiBuka()) { toast(mesejTutup()); tunjukPopup(); return; }
    troli[id] = (troli[id] || 0) + 1;
    kemasKiniUI();
    toast('✅ Ditambah ke troli');
}
function kurangItem(id) {
    if (troli[id] > 1) { troli[id]--; }
    else { delete troli[id]; }
    kemasKiniUI();
}
function clearCart() {
    troli = {};
    kemasKiniUI();
    toast('🗑️ Troli dikosongkan');
}
function jumlahItem() {
    return Object.values(troli).reduce((a, b) => a + b, 0);
}
function jumlahHarga() {
    return Object.entries(troli).reduce((t, [id, q]) => {
        const m = menuData.find(x => x.id === id);
        return t + (m ? m.harga * q : 0);
    }, 0);
}

function binaMenuHTML(m) {
    const qty = troli[m.id] || 0;
    let butang;
    if (!kedaiBuka()) {
        butang = `<button class="add-btn" style="background:#EF4444; opacity:0.5; pointer-events:none;">🔒</button>`;
    } else if (qty > 0) {
        butang = `<div class="quantity-control">
            <button onclick="kurangItem('${m.id}')">−</button>
            <span class="qty-num">${qty}</span>
            <button onclick="tambahItem('${m.id}')">+</button>
        </div>`;
    } else {
        butang = `<button class="add-btn" onclick="tambahItem('${m.id}')">+</button>`;
    }
    
    //  GAMBAR DENGAN CLICK EVENT UNTUK LIGHTBOX
    const imgHTML = m.gambar ? `<img src="${m.gambar}" class="menu-item-img" alt="${m.nama}" onclick="openLightbox('${m.gambar}', '${m.nama}')" style="cursor: pointer;">` : '';
    
    return `
        <div class="menu-item ${m.featured ? 'featured' : ''}">
            ${imgHTML}
            <div class="menu-info">
                <div class="menu-name">${m.nama}</div>
                <div class="menu-desc">${m.desc}</div>
            </div>
            <div class="menu-actions">
                <span class="menu-price">RM${m.harga.toFixed(2)}</span>
                ${butang}
            </div>
        </div>
    `;
}

function kemasKiniUI() {
    const el = document.getElementById('menu-list');
    if (el) el.innerHTML = menuData.map(m => binaMenuHTML(m)).join('');
    const cartEl = document.getElementById('cart-items');
    const totalEl = document.getElementById('cart-total');
    const priceEl = document.getElementById('total-price');
    const clearBtn = document.getElementById('clear-btn');
    const badge = document.getElementById('nav-badge');
    const jml = jumlahItem();
    if (!kedaiBuka()) {
        if (cartEl) cartEl.innerHTML = `<div style="text-align:center; padding:20px; color:#EF4444; font-weight:700; font-size:0.9rem;">${mesejTutup()}</div>`;
        if (totalEl) totalEl.style.display = 'none';
        if (clearBtn) clearBtn.style.display = 'none';
        if (badge) badge.style.display = 'none';
    } else if (jml === 0) {
        if (cartEl) cartEl.innerHTML = '<div class="cart-empty">Tiada item</div>';
        if (totalEl) totalEl.style.display = 'none';
        if (clearBtn) clearBtn.style.display = 'none';
        if (badge) badge.style.display = 'none';
    } else {
        if (cartEl) {
            cartEl.innerHTML = Object.entries(troli).map(([id, q]) => {
                const m = menuData.find(x => x.id === id);
                return m ? `<div class="cart-item-row">
                    <span class="cart-item-name">${m.nama} <span class="cart-item-qty">x${q}</span></span>
                    <span style="font-weight:600;">RM${(m.harga * q).toFixed(2)}</span>
                </div>` : '';
            }).join('');
        }
        if (totalEl) totalEl.style.display = 'flex';
        if (priceEl) priceEl.textContent = `RM${jumlahHarga().toFixed(2)}`;
        if (clearBtn) clearBtn.style.display = 'block';
        if (badge) { badge.style.display = 'flex'; badge.textContent = jml; }
    }
    const orderBtn = document.getElementById('orderBtn');
    if (orderBtn) {
        if (!kedaiBuka()) {
            orderBtn.textContent = ' ' + mesejTutup();
            orderBtn.style.background = '#EF4444';
            orderBtn.style.opacity = '0.7';
        } else {
            orderBtn.textContent = '💬 Order WhatsApp';
            orderBtn.style.background = '#25D366';
            orderBtn.style.opacity = '1';
        }
    }
}

// ==============================================
// 📱 WHATSAPP & NAVIGATION
// ==============================================
function orderWhatsApp() {
    if (!kedaiBuka()) { toast(mesejTutup()); tunjukPopup(); return; }
    const items = Object.entries(troli);
    if (items.length === 0) { toast('🛒 Troli kosong!'); return; }
    let teks = 'Assalamualaikum/ Salam sejahtera Kak Zila, saya nak order:%0A%0A';
    items.forEach(([id, q]) => {
        const m = menuData.find(x => x.id === id);
        if (m) teks += `${m.nama} x${q} = RM${(m.harga * q).toFixed(2)}%0A`;
    });
    teks += `%0A*Jumlah: RM${jumlahHarga().toFixed(2)}*`;
    window.location.href = `https://wa.me/601111640776?text=${teks}`;
}
function bukaWhatsApp() {
    if (!kedaiBuka()) { toast(mesejTutup()); tunjukPopup(); return; }
    window.location.href = 'https://wa.me/601111640776';
}
function bukaMaps() { window.location.href = 'https://maps.google.com/?q=PPR+Sri+Pantai+Blok+102'; }
function bukaWaze() { window.location.href = 'https://waze.com/ul?q=PPR+Sri+Pantai+Blok+102'; }

function tukarTab(nama) {
    document.querySelectorAll('.tab-page').forEach(p => p.classList.remove('active'));
    document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
    const target = document.getElementById('tab-' + nama);
    if (target) target.classList.add('active');
    const labelMap = { menu:'Utama', kad:'Kad niaga', hubungi:'Hubungi', lokasi:'Lokasi' };
    document.querySelectorAll('.nav-item').forEach(n => {
        if (n.textContent.trim().includes(labelMap[nama])) n.classList.add('active');
    });
    document.querySelector('.content').scrollTop = 0;
    kemasKiniUI();
}

// ==============================================
// 💳 KAD BISNES
// ==============================================
function showCard(side, btn) {
    document.querySelectorAll('.bizcard-tab').forEach(t => t.classList.remove('active'));
    btn.classList.add('active');
    document.getElementById('card-front').classList.remove('show');
    document.getElementById('card-back').classList.remove('show');
    setTimeout(() => document.getElementById('card-' + side).classList.add('show'), 50);
}
async function simpanKad() {
    const active = document.querySelector('.bizcard-tab.active');
    const side = active.textContent.includes('Depan') ? 'front' : 'back';
    const url = side === 'front' ? 'https://i.postimg.cc/vZ2gCR74/media-(2).jpg' : 'https://i.postimg.cc/Pxf94qbN/media(3).png';
    try {
        const r = await fetch(url); const b = await r.blob(); const u = URL.createObjectURL(b);
        const a = document.createElement('a'); a.href = u; a.download = `Kad-Kak-Zila-${side}.jpg`; a.click();
        URL.revokeObjectURL(u); toast('✅ Disimpan!');
    } catch(e) { window.open(url, '_blank'); toast('📎 Dibuka di tab baru'); }
}
async function kongsiKad() {
    const d = { title: 'Nasi Lemak Kak Zila', text: 'Jom singgah! PPR Sri Pantai Blok 102. WhatsApp: 011-1164 0776' };
    if (navigator.share) { try { await navigator.share(d); } catch(e) {} }
    else { try { await navigator.clipboard.writeText(d.text); toast('📋 Disalin!'); } catch(e) { toast('📤 Kongsi manual'); } }
}

// ==============================================
// 🔔 TOAST & POPUP
// ==============================================
function toast(msg) {
    const t = document.getElementById('toast');
    t.textContent = msg; t.classList.add('show');
    clearTimeout(t._t); t._t = setTimeout(() => t.classList.remove('show'), 2500);
}
function tunjukPopup() {
    const popup = document.getElementById('popupTutup');
    const popupTitle = popup.querySelector('.popup-title');
    const popupText = document.getElementById('popupText');
    if (ownerMode === 'TUTUP') {
        popupTitle.textContent = '🔴 Kedai Ditutup Oleh Owner';
        popupText.innerHTML = 'Maaf, kedai sedang ditutup oleh owner.<br><br>Sila datang lagi nanti. Terima kasih! 🙏';
    } else if (isHariTutup()) {
        popupTitle.textContent = `🔴 Maaf, Kedai Tutup Hari ${NAMA_HARI[new Date().getDay()]}`;
        const bukaStr = formatTime12(operatingHours.buka);
        const tutupStr = formatTime12(operatingHours.tutup);
        const hariBuka = [];
        for (let i = 0; i < 7; i++) {
            if (!operatingHours.hari_tutup.includes(i)) hariBuka.push(NAMA_HARI[i]);
        }
        popupText.innerHTML = `📅 Hari ini <strong>${NAMA_HARI[new Date().getDay()]}</strong> kedai berehat.<br>🕐 Waktu operasi: <strong>${bukaStr} - ${tutupStr}</strong><br>📆 Hari buka: <strong>${hariBuka.join(', ')}</strong>`;
    } else if (!isWaktuOperasi()) {
        popupTitle.textContent = '🕐 Kedai Belum Dibuka';
        const bukaStr = formatTime12(operatingHours.buka);
        const tutupStr = formatTime12(operatingHours.tutup);
        popupText.innerHTML = `🕐 Waktu operasi kami:<br><strong>${bukaStr} - ${tutupStr}</strong><br><br>Sila datang semula pada waktu operasi. Terima kasih! 🥰`;
    }
    popup.classList.add('show');
}
function tutupPopup() { document.getElementById('popupTutup').classList.remove('show'); }

// ==============================================
// 📤 FUNGSI SHARE SOCIAL MEDIA (LENGKAP)
// ==============================================

const SHARE_URL = 'https://nasi-lemak-kak-zila.pages.dev';
const SHARE_TITLE = '🍽️ Nasi Lemak Kak Zila';
const SHARE_TEXT = 'Jom singgah! Nasi Lemak sedap di PPR Sri Pantai Blok 102 Kuala Lumpur.';

// ==============================================
// 💬 WHATSAPP
// ==============================================
function shareWhatsApp() {
    const teks = `${SHARE_TITLE}%0A%0A${SHARE_TEXT}%0A%0A🔗 ${SHARE_URL}`;
    window.open(`https://wa.me/?text=${teks}`, '_blank');
}

// ==============================================
// ✈️ TELEGRAM
// ==============================================
function shareTelegram() {
    window.open(`https://t.me/share/url?url=${encodeURIComponent(SHARE_URL)}&text=${encodeURIComponent(SHARE_TITLE + '\n\n' + SHARE_TEXT)}`, '_blank');
}
    
// ==============================================
// 📘 FACEBOOK
// ==============================================
    function shareFacebook() {
    const isMobile = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent);
    
    // Copy URL sahaja (bukan teks penuh) - supaya kad preview keluar
    if (navigator.clipboard) {
        navigator.clipboard.writeText(SHARE_URL).then(function() {
            toast('📋 URL disalin!');
            showFacebookGuide();
        });
    }
    
    if (isMobile) {
        // Cuba buka Facebook App
        setTimeout(function() {
            window.location.href = 'fb://composer';
            
            // Fallback: buka Facebook web
            setTimeout(function() {
                window.open('https://m.facebook.com/composer', '_blank');
            }, 1500);
        }, 500);
        
    } else {
        // Desktop: Buka Facebook Share Dialog
        window.open(
            `https://www.facebook.com/sharer/sharer.php?u=${encodeURIComponent(SHARE_URL)}`,
            'facebook-share',
            'width=626,height=436'
        );
    }
}

// Popup panduan cantik
function showFacebookGuide() {
    const guide = document.createElement('div');
    guide.style.cssText = `
        position: fixed;
        bottom: 80px;
        left: 20px;
        right: 20px;
        background: #1877F2;
        color: #fff;
        padding: 20px;
        border-radius: 20px;
        z-index: 999;
        text-align: center;
        font-size: 0.85rem;
        font-weight: 600;
        box-shadow: 0 10px 30px rgba(0,0,0,0.3);
        animation: popIn 0.3s ease;
    `;
    guide.innerHTML = `
        <div style="font-size:2rem; margin-bottom:8px;">📘</div>
        <strong style="font-size:1rem;">Facebook Dibuka!</strong><br><br>
        <span style="font-size:0.8rem; opacity:0.9;">📋 URL dah disalin</span><br><br>
        <span style="background:#fff; color:#1877F2; padding:6px 14px; border-radius:10px; font-size:0.75rem; display:inline-block; margin:8px 0;">
            ${SHARE_URL}
        </span><br><br>
        <span style="font-size:0.75rem; opacity:0.9;">⚡ <strong>Paste URL ini</strong> di post Facebook<br>untuk keluarkan kad preview</span><br>
        <span style="font-size:0.65rem; opacity:0.7;">👇 Tekan untuk tutup</span>
    `;
    guide.onclick = function() { 
        guide.style.opacity = '0';
        guide.style.transition = 'opacity 0.3s';
        setTimeout(function() { guide.remove(); }, 300);
    };
    document.body.appendChild(guide);
    
    // Auto tutup selepas 8 saat
    setTimeout(function() { 
        if (document.body.contains(guide)) {
            guide.style.opacity = '0';
            guide.style.transition = 'opacity 0.3s';
            setTimeout(function() { guide.remove(); }, 300);
        }
    }, 8000);
}
// ==============================================
// 📋 COPY LINK
// ==============================================
function shareCopyLink() {
    const teks = `${SHARE_TITLE}\n\n${SHARE_TEXT}\n\n🔗 ${SHARE_URL}`;
    
    if (navigator.clipboard) {
        navigator.clipboard.writeText(teks).then(function() {
            toast('📋 Pautan & teks disalin ke clipboard!');
        }).catch(function() {
            toast('📤 Sila kongsi manual');
        });
    } else {
        // Fallback untuk older browser
        const textarea = document.createElement('textarea');
        textarea.value = teks;
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
        toast('📋 Pautan & teks disalin!');
    }
}

// ==============================================
// 📱 NATIVE SHARE (Mobile only)
// ==============================================
function shareNative() {
    if (navigator.share) {
        navigator.share({
            title: SHARE_TITLE,
            text: SHARE_TEXT,
            url: SHARE_URL
        }).then(function() {
            toast('✅ Berjaya dikongsi!');
        }).catch(function(error) {
            if (error.name !== 'AbortError') {
                toast('📤 Sila kongsi manual');
            }
        });
    } else {
        shareCopyLink();
    }
}


// ==============================================
// 🤖 AI CHATBOT
// ==============================================
function toggleChat() { document.getElementById('sitiChatBox').classList.toggle('show'); }
function playSendSound() {
    try {
        const ctx = new (window.AudioContext || window.webkitAudioContext)();
        const o = ctx.createOscillator(); const g = ctx.createGain();
        o.connect(g); g.connect(ctx.destination);
        o.type = 'sine'; o.frequency.setValueAtTime(800, ctx.currentTime);
        o.frequency.exponentialRampToValueAtTime(1200, ctx.currentTime + 0.1);
        g.gain.setValueAtTime(0.3, ctx.currentTime);
        g.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.15);
        o.start(ctx.currentTime); o.stop(ctx.currentTime + 0.15);
    } catch(e) {}
}
function playReceiveSound() {
    try {
        const ctx = new (window.AudioContext || window.webkitAudioContext)();
        const o = ctx.createOscillator(); const g = ctx.createGain();
        o.connect(g); g.connect(ctx.destination);
        o.type = 'sine'; o.frequency.setValueAtTime(1000, ctx.currentTime);
        o.frequency.setValueAtTime(1200, ctx.currentTime + 0.08);
        o.frequency.setValueAtTime(1400, ctx.currentTime + 0.16);
        g.gain.setValueAtTime(0.3, ctx.currentTime);
        g.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.3);
        o.start(ctx.currentTime); o.stop(ctx.currentTime + 0.3);
    } catch(e) {}
}
async function sendMessage() {
    if (isSitiReplying) { toast('⏳ Siti AI sedang menulis...'); return; }
    const input = document.getElementById('sitiInput');
    const message = input.value.trim();
    if (!message) return;
    const body = document.getElementById('sitiChatBody');
    const sendBtn = document.querySelector('.siti-chat-input button');
    isSitiReplying = true;
    sendBtn.disabled = true; sendBtn.style.opacity = '0.5'; sendBtn.style.pointerEvents = 'none';
    input.disabled = true; input.style.background = '#F3F4F6';
    body.innerHTML += `<div class="chat-bubble chat-user">${escapeHTML(message)}</div>`;
    input.value = ''; input.blur();
    playSendSound();
    body.scrollTop = body.scrollHeight;
    const typingId = 'typing-' + Date.now();
    body.innerHTML += `<div class="chat-bubble chat-ai" id="${typingId}">✏️ Siti Ai menulis...</div>`;
    body.scrollTop = body.scrollHeight;
    try {

// 📌 backend-ai 📌 
        const res = await fetch('https://zila-food-aigo.onrender.com/api/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message, session_id: sessionId })
        });
        const data = await res.json();
        const typingEl = document.getElementById(typingId);
        if (typingEl) typingEl.remove();
        body.innerHTML += `<div class="chat-bubble chat-ai">${formatAIResponse(data.response)}</div>`;
        playReceiveSound();
        body.scrollTop = body.scrollHeight;
    } catch(e) {
        const typingEl = document.getElementById(typingId);
        if (typingEl) typingEl.remove();
        body.innerHTML += `<div class="chat-bubble chat-ai">😔 Maaf, Siti AI offline sekejap.</div>`;
        body.scrollTop = body.scrollHeight;
    }
    isSitiReplying = false;
    sendBtn.disabled = false; sendBtn.style.opacity = '1'; sendBtn.style.pointerEvents = 'auto';
    input.disabled = false; input.style.background = '#fff';
}

// ==============================================
// 🎨 FORMAT AI RESPONSE (SUPPORT MARKDOWN)
// ==============================================
function formatAIResponse(text) {
    text = escapeHTML(text);
    text = text.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
    text = text.replace(/__(.*?)__/g, '<strong>$1</strong>');
    text = text.replace(/\*(.*?)\*/g, '<em>$1</em>');
    text = text.replace(/_(.*?)_/g, '<em>$1</em>');
    text = text.replace(/^### (.*$)/gm, '<h3 style="font-size:0.9rem; margin:8px 0 4px; color:#2E7D32;">$1</h3>');
    text = text.replace(/^## (.*$)/gm, '<h2 style="font-size:0.95rem; margin:10px 0 4px; color:#2E7D32;">$1</h2>');
    text = text.replace(/^# (.*$)/gm, '<h1 style="font-size:1rem; margin:12px 0 4px; color:#2E7D32;">$1</h1>');
    text = text.replace(/`(.*?)`/g, '<code style="background:#E5E7EB; padding:2px 6px; border-radius:4px; font-size:0.8rem;">$1</code>');
    text = text.replace(/^[-*]\s+(.*$)/gm, '<li style="margin-left:16px; list-style:disc;">$1</li>');
    text = text.replace(/^\d+\.\s+(.*$)/gm, '<li style="margin-left:16px; list-style:decimal;">$1</li>');
    text = text.replace(/^(---|\*\*\*)$/gm, '<hr style="border:0; border-top:1px solid #E5E7EB; margin:8px 0;">');
    text = text.replace(/^>\s+(.*$)/gm, '<blockquote style="border-left:3px solid #2E7D32; padding-left:12px; margin:8px 0; color:#6B7280;">$1</blockquote>');
    text = text.replace(/\[(.*?)\]\((.*?)\)/g, '<a href="$2" target="_blank" style="color:#2563EB; text-decoration:underline;">$1</a>');
    text = text.replace(/~~(.*?)~~/g, '<del>$1</del>');
    text = text.replace(/\n/g, '<br>');
    return text;
}

// ==============================================
// 🛡️ ESCAPE HTML (Security)
// ==============================================
function escapeHTML(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// ==============================================
// 🔽 SCROLL TO BOTTOM - AI CHATBOT
// ==============================================
document.addEventListener('DOMContentLoaded', function() {
    const chatBody = document.getElementById('sitiChatBody');
    const scrollBtn = document.getElementById('scrollBottomBtn');
    
    if (chatBody && scrollBtn) {
        chatBody.addEventListener('scroll', function() {
            const scrollTop = chatBody.scrollTop;
            const scrollHeight = chatBody.scrollHeight;
            const clientHeight = chatBody.clientHeight;
            
            if (scrollHeight - scrollTop - clientHeight > 200) {
                scrollBtn.classList.add('show');
            } else {
                scrollBtn.classList.remove('show');
            }
        });
    }
});

function scrollToBottom() {
    const chatBody = document.getElementById('sitiChatBody');
    const scrollBtn = document.getElementById('scrollBottomBtn');
    
    if (chatBody) {
        chatBody.scrollTo({
            top: chatBody.scrollHeight,
            behavior: 'smooth'
        });
    }
    
    if (scrollBtn) {
        scrollBtn.classList.remove('show');
    }
}

// ==============================================
// 🔐 ADMIN LOGIN
// ==============================================
try { adminToken = localStorage.getItem('zila_admin_token'); } catch(e) {}

function bukaAdminLogin() {
    if (adminToken) { verifyTokenBukaPanel(); return; }
    tunjukPopupLogin();
}
function tunjukPopupLogin() {
    const old = document.getElementById('adminLoginPopup');
    if (old) old.remove();
    const popup = document.createElement('div');
    popup.id = 'adminLoginPopup';
    popup.style.cssText = 'position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.85); z-index:99999; display:flex; align-items:center; justify-content:center;';
    popup.innerHTML = `
        <div style="background:#fff; border-radius:20px; padding:24px; width:90%; max-width:360px; text-align:center;">
            <div style="font-size:2.5rem; margin-bottom:8px;">🔐</div>
            <h3 style="margin-bottom:4px; color:#2E7D32;">Admin Zila Food</h3>
            <p style="font-size:0.8rem; color:#6B7280; margin-bottom:16px;">Masukkan nombor & kata laluan</p>
            <input type="tel" id="adminPhone" placeholder="No. WhatsApp" style="width:100%; padding:12px; border:1px solid #ddd; border-radius:10px; margin-bottom:10px; font-family:inherit; font-size:0.9rem;">
            <input type="password" id="adminPass" placeholder="Kata Laluan" style="width:100%; padding:12px; border:1px solid #ddd; border-radius:10px; margin-bottom:16px; font-family:inherit; font-size:0.9rem;">
            <p id="loginMsg" style="font-size:0.75rem; color:#EF4444; margin-bottom:12px; display:none;"></p>
            <button onclick="loginAdmin()" style="width:100%; padding:14px; background:#2E7D32; color:#fff; border:none; border-radius:12px; font-weight:700; font-size:0.9rem; cursor:pointer;">🔓 Masuk</button>
            <button onclick="document.getElementById('adminLoginPopup').remove()" style="width:100%; padding:10px; background:transparent; border:none; color:#999; margin-top:8px; font-size:0.8rem; cursor:pointer;">❌ Batal</button>
        </div>
    `;
    document.body.appendChild(popup);
}
async function loginAdmin() {
    const phone = document.getElementById('adminPhone').value.trim();
    const password = document.getElementById('adminPass').value.trim();
    const msg = document.getElementById('loginMsg');
    if (!phone || !password) { msg.textContent = '❌ Sila isi nombor dan password.'; msg.style.display = 'block'; return; }
    msg.textContent = '⏳ Sedang login...'; msg.style.color = '#6B7280'; msg.style.display = 'block';
    try {
        const res = await fetch('https://zila-food.onrender.com/api/admin/login', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ phone, password })
        });
        const data = await res.json();
        if (data.success) {
            adminToken = data.token;
            try { localStorage.setItem('zila_admin_token', adminToken); } catch(e) {}
            msg.textContent = '✅ Berjaya!'; msg.style.color = '#2E7D32';
            setTimeout(() => { document.getElementById('adminLoginPopup').remove(); bukaPanelAdmin(); }, 800);
        } else { msg.textContent = data.message; msg.style.color = '#EF4444'; }
    } catch(e) { msg.textContent = '❌ Gagal sambung ke server.'; msg.style.color = '#EF4444'; }
}
async function verifyTokenBukaPanel() {
    try {
        const res = await fetch('https://zila-food.onrender.com/api/admin/verify', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token: adminToken })
        });
        const data = await res.json();
        if (data.valid) { bukaPanelAdmin(); }
        else { adminToken = null; try { localStorage.removeItem('zila_admin_token'); } catch(e) {} tunjukPopupLogin(); }
    } catch(e) { tunjukPopupLogin(); }
}

// ==============================================
// 🎛️ PANEL ADMIN
// ==============================================
function bukaPanelAdmin() {
    document.getElementById('adminOverlay').classList.add('show');
    loadCurrentSettings();
    loadMenuAdmin();
    loadWaktuOperasiAdmin();
}
function tutupPanelAdmin() { document.getElementById('adminOverlay').classList.remove('show'); }

function setMode(mode) {
    ownerMode = mode;
    document.querySelectorAll('.admin-toggle').forEach(btn => btn.classList.remove('active'));
    const modeMap = { 'AUTO': 'modeAuto', 'BUKA': 'modeBuka', 'TUTUP': 'modeTutup' };
    const activeBtn = document.getElementById(modeMap[mode]);
    if (activeBtn) activeBtn.classList.add('active');
    const statusMap = { 'AUTO': '🔄 Auto (Ikut Jadual)', 'BUKA': '🟢 Buka (Override)', 'TUTUP': '🔴 Tutup (Override)' };
    document.getElementById('infoStatus').textContent = statusMap[mode];
    document.getElementById('infoMode').textContent = mode;

    // 🔥 SEMBUNYI/TUNJUK WAKTU OPERASI (Card)
    const cardWaktu = document.getElementById('cardWaktuOperasi');
    if (cardWaktu) {
        if (mode === 'AUTO') {
            cardWaktu.style.display = 'block';
        } else {
            cardWaktu.style.display = 'none';
        }
    }
    
    // 🔥 SEMBUNYI INFO WAKTU BILA BUKA/TUTUP (Info Panel)
    const infoWaktuEl = document.getElementById('infoWaktu');
    if (infoWaktuEl) {
        if (mode === 'AUTO' && operatingHours.buka) {
            infoWaktuEl.parentElement.style.display = 'block';
        } else {
            infoWaktuEl.parentElement.style.display = 'none';
        }
    }
    
    // 🔥 UPDATE DISPLAY WAKTU DI HEADER & LOKASI
    updateWaktuDisplay();
    
    // 🔥 REFRESH UI MENU & TROLI
    kemasKiniUI();
}

// 🔥 LOAD SETTINGS DARI FIREBASE
async function loadCurrentSettings() {
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        const docRef = window.fbDoc(db, 'settings', 'shop_settings');
        const snap = await window.fbGetDoc(docRef);
        let data = { mode: 'AUTO', memo: '', ticker: '', last_updated: null };
        if (snap.exists()) {
            data = snap.data();
        }
        ownerMode = data.mode || 'AUTO';
        currentMemo = data.memo || '';
        currentTicker = data.ticker || '';
        setMode(ownerMode);
        
        // LOAD MEMO
        document.getElementById('adminMemo').value = currentMemo;
        document.getElementById('infoUpdate').textContent = data.last_updated
            ? new Date(data.last_updated).toLocaleString('ms-MY')
            : 'Belum pernah';
        if (currentMemo) {
            document.getElementById('memoPreview').style.display = 'block';
            document.getElementById('memoPreviewText').textContent = currentMemo;
        } else {
            document.getElementById('memoPreview').style.display = 'none';
        }
        
        // 🔥 LOAD TICKER - MACAM MEMO PREVIEW
        const tickerInput = document.getElementById('adminTicker');
        const tickerPreview = document.getElementById('tickerPreview');
        const tickerPreviewText = document.getElementById('tickerPreviewText');
        
        if (tickerInput) {
            tickerInput.value = currentTicker;
        }
        
        if (currentTicker && currentTicker.trim() !== '') {
            if (tickerPreview) {
                tickerPreview.style.display = 'block';
            }
            if (tickerPreviewText) {
                tickerPreviewText.textContent = currentTicker;
            }
        } else {
            if (tickerPreview) {
                tickerPreview.style.display = 'none';
            }
        }
        
        // 🔥 INFO WAKTU - HANYA TUNJUK BILA MODE AUTO
        const infoWaktuEl = document.getElementById('infoWaktu');
        if (infoWaktuEl) {
            if (ownerMode === 'AUTO' && operatingHours.buka) {
                infoWaktuEl.textContent = `${formatTime12(operatingHours.buka)} - ${formatTime12(operatingHours.tutup)}`;
                infoWaktuEl.parentElement.style.display = 'block';
            } else {
                infoWaktuEl.parentElement.style.display = 'none';
            }
        }
    } catch(e) {
        document.getElementById('infoUpdate').textContent = 'Gagal load';
    }
}

// =========================================
// 🔥 SIMPAN SETTINGS KE FIREBASE (mode + memo + ticker)
// =========================================
async function simpanSettings() {
    const memo = document.getElementById('adminMemo').value.trim();
    // 🔥 Ambil nilai ticker (guna ? supaya tak error kalau HTML belum ada)
    const tickerText = document.getElementById('adminTicker')?.value.trim() || ''; 
    
    const msg = document.getElementById('adminMsg');
    if (!adminToken) { msg.textContent = '❌ Sesi tamat. Sila login semula.'; msg.style.color = '#EF4444'; msg.style.display = 'block'; return; }
    msg.textContent = '⏳ Menyimpan...'; msg.style.color = '#6B7280'; msg.style.display = 'block';
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        const docRef = window.fbDoc(db, 'settings', 'shop_settings');
        const now = new Date().toISOString();
        
        await window.fbSetDoc(docRef, {
            mode: ownerMode,
            memo: memo,
            ticker: tickerText, // 🔥 TAMBAH SIMPAN TICKER DI SINI
            last_updated: now
        }, { merge: true });

        currentMemo = memo;
        msg.textContent = '✅ Tetapan disimpan!'; msg.style.color = '#2E7D32';
        document.getElementById('infoUpdate').textContent = new Date(now).toLocaleString('ms-MY');
        
        // 🔥 PREVIEW MEMO
        if (memo) {
            document.getElementById('memoPreview').style.display = 'block';
            document.getElementById('memoPreviewText').textContent = memo;
        } else {
            document.getElementById('memoPreview').style.display = 'none';
        }
        
        // 🔥 PREVIEW TICKER (MACAM MEMO)
        const tickerPreview = document.getElementById('tickerPreview');
        const tickerPreviewText = document.getElementById('tickerPreviewText');
        if (tickerText) {
            tickerPreview.style.display = 'block';
            tickerPreviewText.textContent = tickerText;
        } else {
            tickerPreview.style.display = 'none';
        }
        
        setTimeout(() => { msg.style.display = 'none'; toast('✅ Tetapan dikemaskini!'); }, 1500);
        updateOwnerMemoUI();
        updateTicker(); // 🔥 Update ticker serta-merta lepas simpan
        kemasKiniUI();
    } catch(e) {
        msg.textContent = '❌ Gagal simpan: ' + e.message;
        msg.style.color = '#EF4444';
    }
}

function logoutAdmin() {
    adminToken = null;
    try { localStorage.removeItem('zila_admin_token'); } catch(e) {}
    tutupPanelAdmin();
    toast('👋 Logout berjaya!');
}

// ==============================================
// ⏰ WAKTU OPERASI - ADMIN PANEL FUNCTIONS
// ==============================================
async function loadWaktuOperasiAdmin() {
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        const docRef = window.fbDoc(db, 'settings', 'operating_hours');
        const snap = await window.fbGetDoc(docRef);
        if (snap.exists()) {
            const d = snap.data();
            document.getElementById('adminBuka').value = d.buka || '19:30';
            document.getElementById('adminTutup').value = d.tutup || '00:00';
            const hariTutup = d.hari_tutup || [4];
            document.querySelectorAll('.day-chip').forEach(chip => {
                const day = parseInt(chip.dataset.day);
                if (hariTutup.includes(day)) chip.classList.add('active');
                else chip.classList.remove('active');
            });
        }
    } catch(e) { console.log('Gagal load waktu operasi admin'); }
}
function toggleDay(chip) { chip.classList.toggle('active'); }

async function simpanWaktuOperasi() {
    const msg = document.getElementById('waktuMsg');
    const buka = document.getElementById('adminBuka').value;
    const tutup = document.getElementById('adminTutup').value;
    const hariTutup = [];
    document.querySelectorAll('.day-chip.active').forEach(chip => {
        hariTutup.push(parseInt(chip.dataset.day));
    });
    if (!buka || !tutup) {
        msg.textContent = '❌ Sila isi masa buka dan tutup!';
        msg.style.color = '#EF4444'; msg.style.display = 'block';
        return;
    }
    msg.textContent = '⏳ Menyimpan...'; msg.style.color = '#6B7280'; msg.style.display = 'block';
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        const docRef = window.fbDoc(db, 'settings', 'operating_hours');
        await window.fbSetDoc(docRef, {
            buka: buka,
            tutup: tutup,
            hari_tutup: hariTutup,
            updated_at: new Date().toISOString()
        }, { merge: true });
        operatingHours = { buka, tutup, hari_tutup: hariTutup };
        updateWaktuDisplay();
        kemasKiniUI();
        document.getElementById('infoWaktu').textContent = `${formatTime12(buka)} - ${formatTime12(tutup)}`;
        msg.textContent = '✅ Waktu operasi disimpan!';
        msg.style.color = '#2E7D32';
        setTimeout(() => { msg.style.display = 'none'; toast('✅ Waktu operasi dikemaskini!'); }, 2000);
    } catch(e) {
        msg.textContent = '❌ Gagal simpan!';
        msg.style.color = '#EF4444'; msg.style.display = 'block';
    }
}

// ==============================================
// 🍗 PENGURUSAN MENU (FIREBASE) + IMAGE UPLOAD
// ==============================================
async function loadMenuAdmin() {
    const container = document.getElementById('adminMenuList');
    if (!container) return;
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        const menuRef = window.fbCollection(db, 'menu');
        const snapshot = await window.fbGetDocs(menuRef);
        let html = '';
        snapshot.forEach(docSnap => {
            const m = docSnap.data();
            const imgHTML = m.gambar ? `<img src="${m.gambar}" class="admin-menu-item-img" onerror="this.style.display='none'">` : '';
            html += `
                <div class="admin-menu-item">
                    ${imgHTML}
                    <div class="admin-menu-item-info">
                        <strong>${m.nama}</strong> — RM${parseFloat(m.harga).toFixed(2)} ${m.featured ? '⭐' : ''}
                        <small>${m.desc || 'Tiada deskripsi'}</small>
                    </div>
                    <div class="admin-menu-actions">
                        <button onclick="editMenu('${docSnap.id}')" style="background:#FFC107; color:#1A1A1A;">✏️</button>
                        <button onclick="padamMenu('${docSnap.id}')" style="background:#EF4444; color:#fff;">🗑️</button>
                    </div>
                </div>
            `;
        });
        container.innerHTML = html || '<p style="text-align:center; color:#6B7280; font-size:0.8rem;">Tiada menu. Tambah menu baru!</p>';
    } catch(e) {
        container.innerHTML = '<p style="text-align:center; color:#EF4444; font-size:0.8rem;">Gagal load menu.</p>';
    }
}

function resetImageUploadUI() {
    selectedFile = null;
    finalImageUrl = '';
    document.getElementById('uploadPreviewWrap').classList.remove('show');
    document.getElementById('uploadPlaceholder').style.display = 'block';
    document.getElementById('uploadArea').classList.remove('has-image');
    document.getElementById('menuGambarFile').value = '';
    document.getElementById('menuGambarUrl').value = '';
    hideUploadProgress();
}

function tunjukFormTambah() {
    document.getElementById('menuForm').style.display = 'block';
    document.getElementById('btnTambahMenu').style.display = 'none';
    document.getElementById('menuEditId').value = '';
    document.getElementById('menuExistingGambar').value = '';
    document.getElementById('menuNama').value = '';
    document.getElementById('menuDesc').value = '';
    document.getElementById('menuHarga').value = '';
    document.getElementById('menuFeatured').checked = false;
    resetImageUploadUI();
    
isImageDeleted = false;
}


function batalMenu() {
    document.getElementById('menuForm').style.display = 'none';
    document.getElementById('btnTambahMenu').style.display = 'block';
    resetImageUploadUI();
}

async function editMenu(docId) {
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        const docRef = window.fbDoc(db, 'menu', docId);
        const docSnap = await window.fbGetDoc(docRef);
        const m = docSnap.data();

        isImageDeleted = false;
        document.getElementById('menuForm').style.display = 'block';
        document.getElementById('btnTambahMenu').style.display = 'none';
        document.getElementById('menuEditId').value = docId;
        document.getElementById('menuNama').value = m.nama || '';
        document.getElementById('menuDesc').value = m.desc || '';
        document.getElementById('menuHarga').value = m.harga || '';
        document.getElementById('menuFeatured').checked = m.featured || false;
        resetImageUploadUI();
        
        if (m.gambar) {
            finalImageUrl = m.gambar;
            const previewImg = document.getElementById('uploadPreviewImg');
            const previewWrap = document.getElementById('uploadPreviewWrap');
            const placeholder = document.getElementById('uploadPlaceholder');
            const uploadArea = document.getElementById('uploadArea');
            previewImg.src = m.gambar;
            previewWrap.classList.add('show');
            placeholder.style.display = 'none';
            uploadArea.classList.add('has-image');
            
            let pathToSave = m.gambarPath || '';
            
            if (!pathToSave && m.gambar.includes('res.cloudinary.com')) {
                try {
                    const urlParts = m.gambar.split('/upload/');
                    if (urlParts.length > 1) {
                        const versionAndId = urlParts[1];
                        const publicId = versionAndId.replace(/v\d+\//, '').replace(/\.[^/.]+$/, "");
                        pathToSave = 'cloudinary:' + publicId;
                        
                        window.fbSetDoc(window.fbDoc(db, 'menu', docId), { gambarPath: pathToSave }, { merge: true });
                        console.log('✅ Auto-save gambarPath ke Firestore:', pathToSave);
                    }
                } catch(e) { console.log('Gagal extract public_id', e); }
            }
            
            document.getElementById('menuExistingGambar').value = pathToSave;
            document.getElementById('uploadFileName').textContent = 'Cloudinary';
            document.getElementById('uploadFileSize').textContent = pathToSave ? 'Ada' : '-';
        } else {
            document.getElementById('menuExistingGambar').value = '';
        }
    } catch(e) { 
        console.error('Gagal load menu!', e); 
        alert('Gagal load menu!'); 
    }
}

async function simpanMenu() {
    if (isUploading) { toast('⏳ Sedang memuat naik gambar...'); return; }
    const msg = document.getElementById('menuMsg');
    const editId = document.getElementById('menuEditId').value;
    const existingGambarPath = document.getElementById('menuExistingGambar').value;
    const btnSimpan = document.getElementById('btnSimpanMenu');
    const data = {
        nama: document.getElementById('menuNama').value.trim(),
        desc: document.getElementById('menuDesc').value.trim(),
        harga: parseFloat(document.getElementById('menuHarga').value) || 0,
        featured: document.getElementById('menuFeatured').checked,
        aktif: true
    };
    if (!data.nama || !data.harga) {
        msg.textContent = '❌ Nama dan harga wajib diisi!';
        msg.style.color = '#EF4444'; msg.style.display = 'block';
        return;
    }
    btnSimpan.disabled = true;
    btnSimpan.textContent = '⏳ Menyimpan...';
    btnSimpan.style.opacity = '0.7';
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        let gambarURL = '';
        let gambarPath = '';
        let needDeleteOld = false;


                        if (selectedFile) {
            isUploading = true;
            showUploadProgress(10, 'Memampatkan gambar...');
            try {
                const tempId = editId || 'temp_' + Date.now();
                showUploadProgress(30, 'Memuat naik ke server...');
                const result = await uploadImageToStorage(selectedFile, tempId);
                showUploadProgress(90, 'Selesai! Menyimpan data...');
                
                gambarURL = result.url;
                gambarPath = result.path;
                
                if (editId && existingGambarPath && (existingGambarPath.startsWith('cloudinary:') || existingGambarPath.startsWith('imgbb:'))) {
                    needDeleteOld = true;
                }
                
                showUploadProgress(100, '✅ Berjaya!');
            } catch(uploadErr) {
                console.error('Upload error:', uploadErr);
                msg.textContent = '❌ Gagal muat naik gambar: ' + uploadErr.message;
                msg.style.color = '#EF4444'; msg.style.display = 'block';
                btnSimpan.disabled = false; btnSimpan.textContent = '💾 Simpan'; btnSimpan.style.opacity = '1';
                hideUploadProgress(); isUploading = false; return;
            }
            isUploading = false;
            setTimeout(hideUploadProgress, 1500);
            
        } else if (finalImageUrl) {
            gambarURL = finalImageUrl;
            gambarPath = 'url_luar:' + finalImageUrl;
            
            if (editId && existingGambarPath && (existingGambarPath.startsWith('cloudinary:') || existingGambarPath.startsWith('imgbb:'))) {
                needDeleteOld = true;
            }
            
        } else if (isImageDeleted) {
            console.log('🗑️ User tekan butang X. existingGambarPath:', existingGambarPath);
            gambarURL = '';
            gambarPath = '';
            
            if (editId && existingGambarPath && (existingGambarPath.startsWith('cloudinary:') || existingGambarPath.startsWith('imgbb:'))) {
                needDeleteOld = true;
                console.log('✅ needDeleteOld = true!');
            } else {
                console.log('⚠️ needDeleteOld = false. existingGambarPath:', existingGambarPath);
            }
            
        } else if (editId) {
            const existingMenu = menuData.find(m => m.id === editId);
            if (existingMenu) {
                gambarURL = existingMenu.gambar || '';
                gambarPath = existingGambarPath || existingMenu.gambarPath || '';
            }
        }


        data.gambar = gambarURL;
        data.gambarPath = gambarPath;


        if (editId) {
            await window.fbSetDoc(window.fbDoc(db, 'menu', editId), data, { merge: true });
        } else {
            const newDocRef = await window.fbAddDoc(window.fbCollection(db, 'menu'), data);
            if (selectedFile && gambarPath) {
                try {
                    const correctResult = await uploadImageToStorage(selectedFile, newDocRef.id);
                    await window.fbSetDoc(newDocRef, {
                        gambar: correctResult.url,
                        gambarPath: correctResult.path
                    }, { merge: true });
                    await deleteImageFromStorage(gambarPath);
                } catch(e) {
                    console.log('Gagal rename file, guna temporary path:', e);
                }
            }
        }

        if (needDeleteOld && existingGambarPath) {
            deleteImageFromStorage(existingGambarPath);
        }

        msg.textContent = '✅ Menu disimpan!';
        msg.style.color = '#2E7D32'; msg.style.display = 'block';
        btnSimpan.disabled = false;
        btnSimpan.textContent = '💾 Simpan';
        btnSimpan.style.opacity = '1';
        batalMenu();
        loadMenuAdmin();
        await fetchMenu();
        kemasKiniUI();
        setTimeout(() => { msg.style.display = 'none'; }, 2000);
        toast('✅ Menu berjaya disimpan!');
    } catch(e) {
        console.error('Simpan error:', e);
        msg.textContent = '❌ Gagal simpan: ' + e.message;
        msg.style.color = '#EF4444'; msg.style.display = 'block';
        btnSimpan.disabled = false;
        btnSimpan.textContent = '💾 Simpan';
        btnSimpan.style.opacity = '1';
        isUploading = false;
        hideUploadProgress();
    }
}

async function padamMenu(docId) {
    if (!confirm('Padam menu ini? Gambar juga akan dipadam dari storage.')) return;
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        const docRef = window.fbDoc(db, 'menu', docId);
        const docSnap = await window.fbGetDoc(docRef);
        const menuData_item = docSnap.data();
        const gambarPath = menuData_item.gambarPath || '';
        await window.fbDeleteDoc(docRef);
        if (gambarPath) deleteImageFromStorage(gambarPath);
        loadMenuAdmin();
        await fetchMenu();
        kemasKiniUI();
        toast('🗑️ Menu & gambar dipadam!');
    } catch(e) {
        console.error('Padam error:', e);
        alert('Gagal padam!');
    }
}

// ==============================================
// 📝 UPDATE MEMO UI (dari Firebase)
// ==============================================
function updateOwnerMemoUI() {
    const memoEl = document.getElementById('ownerMemo');
    if (!memoEl) return;
    if (ownerMode === 'TUTUP') {
        memoEl.textContent = '🔴 ' + (currentMemo || 'Kedai ditutup buat sementara.');
        memoEl.style.display = 'block';
        memoEl.style.background = '#FEF2F2';
        memoEl.style.color = '#DC2626';
    } else if (ownerMode === 'BUKA') {
        memoEl.textContent = '🟢 ' + (currentMemo || 'Kedai dibuka khas!');
        memoEl.style.display = 'block';
        memoEl.style.background = '#F0FDF4';
        memoEl.style.color = '#16A34A';
    } else if (currentMemo) {
        memoEl.textContent = 'ℹ️ ' + currentMemo;
        memoEl.style.display = 'block';
        memoEl.style.background = '#EFF6FF';
        memoEl.style.color = '#1E40AF';
    } else {
        memoEl.style.display = 'none';
    }
}

// ==============================================
// 🚀 INIT
// ==============================================
async function initApp() {
    await Promise.all([
        fetchWaktuOperasi(),
        fetchMenu(),
        fetchShopSettings()
    ]);
    updateOwnerMemoUI();
    updateTicker(); // 🔥 TAMBAH BARIS INI
    
    if (!kedaiBuka()) {
        setTimeout(tunjukPopup, 1500);
    }
    kemasKiniUI();
    
    setInterval(async () => {
        await Promise.all([fetchWaktuOperasi(), fetchShopSettings()]);
        updateOwnerMemoUI();
        updateTicker(); // 🔥 TAMBAH BARIS INI JUGA
        kemasKiniUI();
    }, 30000);
}

// ==============================================
// 📢 FUNGSI BARU: UPDATE TICKER (Teks Berjalan)
// ==============================================
async function updateTicker() {
    try {
        await waitForFirebase();
        const db = window.firebaseDB;
        const docRef = window.fbDoc(db, 'settings', 'shop_settings');
        const snap = await window.fbGetDoc(docRef);
        if (snap.exists()) {
            const d = snap.data();
            const tickerData = d.ticker || '';
            const tickerContainer = document.getElementById('tickerContainer');
            const tickerTextEl = document.getElementById('tickerText');
            const content = document.querySelector('.content');

            if (tickerContainer && tickerTextEl && content) {
                if (tickerData && tickerData.trim() !== '') {
                    tickerTextEl.textContent = tickerData;
                    tickerContainer.classList.add('show');
                    content.classList.add('has-ticker');
                } else {
                    tickerTextEl.textContent = '';
                    tickerContainer.classList.remove('show');
                    content.classList.remove('has-ticker');
                }
            }
        }
    } catch(e) {
        console.log('Gagal load ticker');
    }
}

initApp();

// ==============================================
// 🖼️ LIGHTBOX GAMBAR MENU
// ==============================================
function openLightbox(imageUrl, caption) {
    const lightbox = document.getElementById('imageLightbox');
    const lightboxImage = document.getElementById('lightboxImage');
    const lightboxCaption = document.getElementById('lightboxCaption');
    
    lightboxImage.src = imageUrl;
    lightboxCaption.textContent = caption || '';
    lightbox.classList.add('show');
    
    // Disable scroll
    document.body.style.overflow = 'hidden';
}

function closeLightbox(event) {
    if (event) {
        event.stopPropagation();
    }
    
    const lightbox = document.getElementById('imageLightbox');
    lightbox.classList.remove('show');
    
    // Enable scroll
    document.body.style.overflow = '';
}

// Close lightbox bila tekan ESC
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
        closeLightbox();
    }
});

initApp();

// FUNGSI TICKER DATA
function previewTicker() {
    const input = document.getElementById('adminTicker');
    const preview = document.getElementById('tickerPreview');
    const previewText = document.getElementById('tickerPreviewText');
    
    if (input.value.trim() !== '') {
        previewText.textContent = input.value;
        preview.style.display = 'block';
    } else {
        preview.style.display = 'none';
    }
}
