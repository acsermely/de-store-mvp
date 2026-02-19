// Federated Storage Network - Web UI JavaScript

const API_BASE_URL = window.location.origin; // Assumes UI is served from same origin as API

let authToken = localStorage.getItem('fsn_token');
let userEmail = localStorage.getItem('fsn_email');
let userCredits = localStorage.getItem('fsn_credits') || 0;

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    if (authToken) {
        showLoggedInState();
        loadFiles();
    }

    // Setup drag and drop
    setupDragAndDrop();
});

// Authentication Functions

async function login() {
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;

    if (!email || !password) {
        showAuthStatus('Please enter email and password', 'error');
        return;
    }

    try {
        showAuthStatus('Logging in...', 'info');

        const response = await fetch(`${API_BASE_URL}/api/v1/auth/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ email, password })
        });

        const data = await response.json();

        if (response.ok) {
            authToken = data.token;
            userEmail = data.email;
            
            // Save to localStorage
            localStorage.setItem('fsn_token', authToken);
            localStorage.setItem('fsn_email', userEmail);
            
            showLoggedInState();
            showAuthStatus('Login successful!', 'success');
            loadFiles();
        } else {
            showAuthStatus(data.error || 'Login failed', 'error');
        }
    } catch (error) {
        showAuthStatus('Network error: ' + error.message, 'error');
    }
}

async function register() {
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;

    if (!email || !password) {
        showAuthStatus('Please enter email and password', 'error');
        return;
    }

    if (password.length < 8) {
        showAuthStatus('Password must be at least 8 characters', 'error');
        return;
    }

    try {
        showAuthStatus('Registering...', 'info');

        const response = await fetch(`${API_BASE_URL}/api/v1/auth/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ email, password })
        });

        const data = await response.json();

        if (response.ok) {
            authToken = data.token;
            userEmail = data.email;
            
            localStorage.setItem('fsn_token', authToken);
            localStorage.setItem('fsn_email', userEmail);
            
            showLoggedInState();
            showAuthStatus('Registration successful! 1,000 credits added.', 'success');
            loadFiles();
        } else {
            showAuthStatus(data.error || 'Registration failed', 'error');
        }
    } catch (error) {
        showAuthStatus('Network error: ' + error.message, 'error');
    }
}

function logout() {
    authToken = null;
    userEmail = null;
    userCredits = 0;
    
    localStorage.removeItem('fsn_token');
    localStorage.removeItem('fsn_email');
    localStorage.removeItem('fsn_credits');
    
    document.getElementById('loginForm').classList.remove('hidden');
    document.getElementById('userInfo').classList.add('hidden');
    document.getElementById('uploadSection').classList.add('hidden');
    document.getElementById('filesSection').classList.add('hidden');
    
    showAuthStatus('Logged out', 'info');
}

function showLoggedInState() {
    document.getElementById('loginForm').classList.add('hidden');
    document.getElementById('userInfo').classList.remove('hidden');
    document.getElementById('uploadSection').classList.remove('hidden');
    document.getElementById('filesSection').classList.remove('hidden');
    
    document.getElementById('userEmail').textContent = userEmail;
    document.getElementById('tokenDisplay').textContent = authToken;
    document.getElementById('creditsAmount').textContent = userCredits;
}

function showAuthStatus(message, type) {
    const status = document.getElementById('authStatus');
    status.textContent = message;
    status.className = `status-message status-${type}`;
    status.style.display = 'block';
    
    setTimeout(() => {
        status.style.display = 'none';
    }, 5000);
}

// File Upload Functions

function setupDragAndDrop() {
    const dropZone = document.getElementById('dropZone');

    dropZone.addEventListener('dragover', (e) => {
        e.preventDefault();
        dropZone.classList.add('dragover');
    });

    dropZone.addEventListener('dragleave', () => {
        dropZone.classList.remove('dragover');
    });

    dropZone.addEventListener('drop', (e) => {
        e.preventDefault();
        dropZone.classList.remove('dragover');
        
        const files = e.dataTransfer.files;
        if (files.length > 0) {
            uploadFile(files[0]);
        }
    });
}

function handleFileSelect(event) {
    const file = event.target.files[0];
    if (file) {
        uploadFile(file);
    }
}

async function uploadFile(file) {
    if (!authToken) {
        showUploadStatus('Please login first', 'error');
        return;
    }

    try {
        showUploadStatus(`Preparing to upload: ${file.name}...`, 'info');
        
        // Show progress bar
        document.getElementById('progressBar').style.display = 'block';
        updateProgress(10);

        // Step 1: Initiate upload
        const initiateResponse = await fetch(`${API_BASE_URL}/api/v1/files/upload/initiate`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`
            },
            body: JSON.stringify({
                filename: file.name,
                size_bytes: file.size,
                mime_type: file.type || 'application/octet-stream'
            })
        });

        updateProgress(30);

        if (!initiateResponse.ok) {
            const error = await initiateResponse.json();
            throw new Error(error.error || 'Failed to initiate upload');
        }

        const uploadSession = await initiateResponse.json();
        console.log('Upload session:', uploadSession);

        // Step 2: Read file and upload chunks
        const chunkSize = uploadSession.chunk_size || 262144; // 256KB default
        const chunks = Math.ceil(file.size / chunkSize);
        
        updateProgress(40);

        // For MVP, we'll upload the whole file as base64 in chunks
        // In production, you'd use FileReader and proper chunking
        for (let i = 0; i < chunks; i++) {
            const start = i * chunkSize;
            const end = Math.min(start + chunkSize, file.size);
            const chunk = file.slice(start, end);
            
            // Convert chunk to base64
            const base64Chunk = await readFileAsBase64(chunk);
            
            const chunkResponse = await fetch(`${API_BASE_URL}/api/v1/files/upload/${uploadSession.session_id}/chunk`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${authToken}`
                },
                body: JSON.stringify({
                    chunk_index: i,
                    data: base64Chunk
                })
            });

            if (!chunkResponse.ok) {
                throw new Error(`Failed to upload chunk ${i}`);
            }

            // Update progress
            const progress = 40 + ((i + 1) / chunks) * 50;
            updateProgress(progress);
        }

        updateProgress(90);

        // Step 3: Complete upload
        const completeResponse = await fetch(`${API_BASE_URL}/api/v1/files/upload/${uploadSession.session_id}/complete`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`
            }
        });

        updateProgress(100);

        if (!completeResponse.ok) {
            const error = await completeResponse.json();
            throw new Error(error.error || 'Failed to complete upload');
        }

        const result = await completeResponse.json();
        
        showUploadStatus(`✅ ${file.name} uploaded successfully! Deducted ${result.credits_deducted} credits.`, 'success');
        
        // Update credits display
        if (result.credits_deducted) {
            userCredits = parseInt(userCredits) - result.credits_deducted;
            document.getElementById('creditsAmount').textContent = userCredits;
        }

        // Refresh file list
        setTimeout(() => loadFiles(), 1000);

        // Reset progress after delay
        setTimeout(() => {
            document.getElementById('progressBar').style.display = 'none';
            updateProgress(0);
        }, 2000);

    } catch (error) {
        console.error('Upload error:', error);
        showUploadStatus('Upload failed: ' + error.message, 'error');
        document.getElementById('progressBar').style.display = 'none';
    }
}

function readFileAsBase64(blob) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => {
            // Remove data URL prefix
            const base64 = reader.result.split(',')[1];
            resolve(base64);
        };
        reader.onerror = reject;
        reader.readAsDataURL(blob);
    });
}

function updateProgress(percent) {
    document.getElementById('progressFill').style.width = percent + '%';
}

function showUploadStatus(message, type) {
    const status = document.getElementById('uploadStatus');
    status.textContent = message;
    status.className = `status-message status-${type}`;
    status.style.display = 'block';
    
    setTimeout(() => {
        status.style.display = 'none';
    }, 5000);
}

// File Listing and Download Functions

async function loadFiles() {
    if (!authToken) {
        return;
    }

    try {
        document.getElementById('filesList').innerHTML = '<p style="color: #888; text-align: center;">Loading files...</p>';

        const response = await fetch(`${API_BASE_URL}/api/v1/files`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Failed to load files');
        }

        const data = await response.json();
        const files = data.files || [];

        if (files.length === 0) {
            document.getElementById('filesList').innerHTML = `
                <p style="color: #888; text-align: center; padding: 30px;">
                    No files yet. Upload your first file above!
                </p>
            `;
            return;
        }

        // Render file list
        let html = '';
        files.forEach(file => {
            const sizeFormatted = formatFileSize(file.size_bytes);
            const dateFormatted = new Date(file.created_at).toLocaleDateString();
            const statusIcon = file.status === 'ready' ? '✅' : '⏳';
            
            html += `
                <div class="file-item">
                    <div class="file-info">
                        <div class="file-icon">${getFileIcon(file.filename)}</div>
                        <div class="file-details">
                            <h4>${statusIcon} ${file.filename}</h4>
                            <p>${sizeFormatted} • ${dateFormatted} • Status: ${file.status}</p>
                        </div>
                    </div>
                    <div class="file-actions">
                        <button class="btn btn-primary btn-small" onclick="downloadFile('${file.id}', '${file.filename}')">
                            ⬇️ Download
                        </button>
                        <button class="btn btn-secondary btn-small" onclick="deleteFile('${file.id}')">
                            🗑️ Delete
                        </button>
                    </div>
                </div>
            `;
        });

        document.getElementById('filesList').innerHTML = html;

    } catch (error) {
        console.error('Load files error:', error);
        showFilesStatus('Failed to load files: ' + error.message, 'error');
    }
}

async function downloadFile(fileId, filename) {
    if (!authToken) {
        showFilesStatus('Please login first', 'error');
        return;
    }

    try {
        showFilesStatus(`Downloading ${filename}...`, 'info');

        const response = await fetch(`${API_BASE_URL}/api/v1/files/${fileId}/download`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Download failed');
        }

        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);

        showFilesStatus(`✅ ${filename} downloaded successfully!`, 'success');

    } catch (error) {
        console.error('Download error:', error);
        showFilesStatus('Download failed: ' + error.message, 'error');
    }
}

async function deleteFile(fileId) {
    if (!confirm('Are you sure you want to delete this file?')) {
        return;
    }

    if (!authToken) {
        showFilesStatus('Please login first', 'error');
        return;
    }

    try {
        const response = await fetch(`${API_BASE_URL}/api/v1/files/${fileId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Delete failed');
        }

        showFilesStatus('✅ File deleted successfully', 'success');
        loadFiles();

    } catch (error) {
        console.error('Delete error:', error);
        showFilesStatus('Delete failed: ' + error.message, 'error');
    }
}

function showFilesStatus(message, type) {
    const status = document.getElementById('filesStatus');
    status.textContent = message;
    status.className = `status-message status-${type}`;
    status.style.display = 'block';
    
    setTimeout(() => {
        status.style.display = 'none';
    }, 5000);
}

// Utility Functions

function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function getFileIcon(filename) {
    const ext = filename.split('.').pop().toLowerCase();
    const icons = {
        'txt': 'TXT',
        'pdf': 'PDF',
        'jpg': 'IMG',
        'jpeg': 'IMG',
        'png': 'IMG',
        'gif': 'IMG',
        'mp4': 'VID',
        'mp3': 'AUD',
        'zip': 'ZIP',
        'doc': 'DOC',
        'docx': 'DOC'
    };
    return icons[ext] || 'FILE';
}