document.addEventListener('DOMContentLoaded', () => {
    const dropZone = document.getElementById('drop-zone');
    const fileInput = document.getElementById('file-input');
    const fileList = document.getElementById('file-list');
    const serverAddress = document.getElementById('server-address');
    const uploadProgress = document.getElementById('upload-progress');
    const progressFill = uploadProgress.querySelector('.progress-fill');
    const progressText = uploadProgress.querySelector('.progress-text');

    // Fetch and display server info
    async function loadServerInfo() {
        try {
            const res = await fetch('/api/info');
            const data = await res.json();
            serverAddress.textContent = `http://${data.ip}:${data.port}`;
        } catch (err) {
            serverAddress.textContent = 'Unable to load';
        }
    }

    // Fetch and display files
    async function loadFiles() {
        try {
            const res = await fetch('/api/files');
            const data = await res.json();
            renderFiles(data.files);
        } catch (err) {
            fileList.innerHTML = '<p class="empty-state">Failed to load files</p>';
        }
    }

    function renderFiles(files) {
        if (!files || files.length === 0) {
            fileList.innerHTML = '<p class="empty-state">No files uploaded yet</p>';
            return;
        }

        fileList.innerHTML = files.map(file => `
            <div class="file-item">
                <div class="file-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                        <polyline points="14 2 14 8 20 8"/>
                    </svg>
                </div>
                <div class="file-info">
                    <div class="file-name">${escapeHtml(file.name)}</div>
                    <div class="file-meta">${formatSize(file.size)} · ${formatDate(file.uploadedAt)}</div>
                </div>
                <div class="file-actions">
                    <a href="/api/download/${file.id}" class="download-btn" download>Download</a>
                    <button class="delete-btn" data-id="${file.id}">Delete</button>
                </div>
            </div>
        `).join('');

        fileList.querySelectorAll('.delete-btn').forEach(btn => {
            btn.addEventListener('click', () => deleteFile(btn.dataset.id));
        });
    }

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    function formatSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    }

    function formatDate(dateStr) {
        const date = new Date(dateStr);
        const now = new Date();
        const diff = now - date;

        if (diff < 60000) return 'Just now';
        if (diff < 3600000) return Math.floor(diff / 60000) + ' min ago';
        if (diff < 86400000) return Math.floor(diff / 3600000) + ' hours ago';

        return date.toLocaleDateString();
    }

    // Upload file
    async function uploadFile(file) {
        const formData = new FormData();
        formData.append('file', file);

        uploadProgress.classList.remove('hidden');
        progressFill.style.width = '0%';
        progressText.textContent = `Uploading ${file.name}...`;

        try {
            const xhr = new XMLHttpRequest();

            xhr.upload.addEventListener('progress', (e) => {
                if (e.lengthComputable) {
                    const percent = (e.loaded / e.total) * 100;
                    progressFill.style.width = percent + '%';
                }
            });

            await new Promise((resolve, reject) => {
                xhr.onload = () => {
                    if (xhr.status === 200) {
                        resolve();
                    } else {
                        reject(new Error('Upload failed'));
                    }
                };
                xhr.onerror = () => reject(new Error('Upload failed'));
                xhr.open('POST', '/api/upload');
                xhr.send(formData);
            });

            progressText.textContent = 'Upload complete!';
            setTimeout(() => {
                uploadProgress.classList.add('hidden');
            }, 1500);

            loadFiles();
        } catch (err) {
            progressText.textContent = 'Upload failed';
            setTimeout(() => {
                uploadProgress.classList.add('hidden');
            }, 2000);
        }
    }

    async function uploadFiles(files) {
        for (const file of files) {
            await uploadFile(file);
        }
    }

    async function deleteFile(id) {
        try {
            const res = await fetch(`/api/delete/${id}`, { method: 'DELETE' });
            if (res.ok) {
                loadFiles();
            }
        } catch (err) {
            console.error('Delete failed:', err);
        }
    }

    // Drag and drop handlers
    dropZone.addEventListener('dragover', (e) => {
        e.preventDefault();
        dropZone.classList.add('drag-over');
    });

    dropZone.addEventListener('dragleave', (e) => {
        e.preventDefault();
        dropZone.classList.remove('drag-over');
    });

    dropZone.addEventListener('drop', (e) => {
        e.preventDefault();
        dropZone.classList.remove('drag-over');
        const files = e.dataTransfer.files;
        if (files.length > 0) {
            uploadFiles(files);
        }
    });

    // Click to upload
    dropZone.addEventListener('click', () => {
        fileInput.click();
    });

    fileInput.addEventListener('change', () => {
        if (fileInput.files.length > 0) {
            uploadFiles(fileInput.files);
            fileInput.value = '';
        }
    });

    // Initial load
    loadServerInfo();
    loadFiles();
});
