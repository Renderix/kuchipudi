/**
 * Kuchipudi Web Interface
 * Vanilla JavaScript application for gesture control management
 */

// API Base URL
const API_BASE = '/api';

// DOM Elements
const navLinks = document.querySelectorAll('.nav-link');
const sections = document.querySelectorAll('.section');
const statusDot = document.getElementById('status-dot');
const statusText = document.getElementById('status-text');
const gestureCount = document.getElementById('gesture-count');
const actionCount = document.getElementById('action-count');
const gestureList = document.getElementById('gesture-list');
const activityList = document.getElementById('activity-list');
const cameraSelect = document.getElementById('camera-select');
const sensitivitySlider = document.getElementById('sensitivity-slider');
const sensitivityValue = document.getElementById('sensitivity-value');
const startAtLogin = document.getElementById('start-at-login');
const saveSettingsBtn = document.getElementById('save-settings-btn');
const addGestureBtn = document.getElementById('add-gesture-btn');

/**
 * API Helper - Fetch JSON from endpoint
 * @param {string} endpoint - API endpoint path
 * @param {object} options - Fetch options
 * @returns {Promise<object>} - Parsed JSON response
 */
async function fetchJSON(endpoint, options = {}) {
    const url = `${API_BASE}${endpoint}`;
    const defaultOptions = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const mergedOptions = { ...defaultOptions, ...options };

    try {
        const response = await fetch(url, mergedOptions);

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        return await response.json();
    } catch (error) {
        console.error(`API Error (${endpoint}):`, error);
        throw error;
    }
}

/**
 * Navigation - Switch between sections
 * @param {string} sectionId - ID of section to show
 */
function navigateTo(sectionId) {
    // Update nav links
    navLinks.forEach(link => {
        if (link.dataset.section === sectionId) {
            link.classList.add('active');
        } else {
            link.classList.remove('active');
        }
    });

    // Update sections
    sections.forEach(section => {
        if (section.id === sectionId) {
            section.classList.add('active');
        } else {
            section.classList.remove('active');
        }
    });

    // Load section-specific data
    if (sectionId === 'dashboard') {
        checkHealth();
        loadGestures();
    } else if (sectionId === 'gestures') {
        loadGestures();
    } else if (sectionId === 'settings') {
        loadCameras();
    }
}

/**
 * Check API health status
 */
async function checkHealth() {
    try {
        const data = await fetchJSON('/health');

        statusDot.classList.remove('offline');
        statusDot.classList.add('online');
        statusText.textContent = 'Online';

        // Update action count if available
        if (data.actions !== undefined) {
            actionCount.textContent = data.actions;
        }
    } catch (error) {
        statusDot.classList.remove('online');
        statusDot.classList.add('offline');
        statusText.textContent = 'Offline';
    }
}

/**
 * Load gestures from API
 */
async function loadGestures() {
    try {
        const data = await fetchJSON('/gestures');

        // API returns { gestures: [...] } format
        const gestures = data.gestures || data;

        // Update gesture count
        gestureCount.textContent = Array.isArray(gestures) ? gestures.length : 0;

        // Render gesture cards
        renderGestureCards(gestures);
    } catch (error) {
        console.error('Failed to load gestures:', error);
        gestureCount.textContent = '0';
        renderEmptyState();
    }
}

/**
 * Render gesture cards in the grid
 * @param {Array} gestures - Array of gesture objects
 */
function renderGestureCards(gestures) {
    if (!Array.isArray(gestures) || gestures.length === 0) {
        renderEmptyState();
        return;
    }

    gestureList.innerHTML = gestures.map(gesture => `
        <div class="gesture-card" data-id="${gesture.id || gesture.name}">
            <h4>${escapeHtml(gesture.name)}</h4>
            <p>${escapeHtml(gesture.description || 'No description')}</p>
            ${gesture.action ? `<span class="gesture-action">${escapeHtml(gesture.action)}</span>` : ''}
        </div>
    `).join('');
}

/**
 * Render empty state for gesture list
 */
function renderEmptyState() {
    gestureList.innerHTML = `
        <div class="empty-state">
            <p>No gestures configured yet.</p>
            <button class="btn btn-primary" onclick="showAddGestureModal()">Add Your First Gesture</button>
        </div>
    `;
}

/**
 * Load available cameras
 */
async function loadCameras() {
    try {
        // Try to get camera list from API first
        const cameras = await fetchJSON('/cameras');
        renderCameraOptions(cameras);
    } catch (error) {
        // Fallback to browser API if available
        if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
            try {
                const devices = await navigator.mediaDevices.enumerateDevices();
                const videoDevices = devices.filter(device => device.kind === 'videoinput');
                renderCameraOptions(videoDevices.map((device, index) => ({
                    id: device.deviceId,
                    name: device.label || `Camera ${index + 1}`
                })));
            } catch (err) {
                console.error('Failed to enumerate devices:', err);
                cameraSelect.innerHTML = '<option value="">No cameras found</option>';
            }
        } else {
            cameraSelect.innerHTML = '<option value="">Camera API unavailable</option>';
        }
    }
}

/**
 * Render camera options in select
 * @param {Array} cameras - Array of camera objects
 */
function renderCameraOptions(cameras) {
    if (!Array.isArray(cameras) || cameras.length === 0) {
        cameraSelect.innerHTML = '<option value="">No cameras found</option>';
        return;
    }

    cameraSelect.innerHTML = cameras.map(camera => `
        <option value="${escapeHtml(camera.id)}">${escapeHtml(camera.name)}</option>
    `).join('');
}

/**
 * Save settings to API
 */
async function saveSettings() {
    const settings = {
        cameraId: cameraSelect.value,
        sensitivity: parseInt(sensitivitySlider.value, 10),
        startAtLogin: startAtLogin.checked
    };

    try {
        await fetchJSON('/settings', {
            method: 'POST',
            body: JSON.stringify(settings)
        });
        alert('Settings saved successfully!');
    } catch (error) {
        console.error('Failed to save settings:', error);
        alert('Failed to save settings. Please try again.');
    }
}

/**
 * Navigate to gesture recording page
 */
function showAddGestureModal() {
    window.location.href = '/record.html';
}

/**
 * Escape HTML to prevent XSS
 * @param {string} text - Text to escape
 * @returns {string} - Escaped text
 */
function escapeHtml(text) {
    if (text === null || text === undefined) {
        return '';
    }
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}

/**
 * Update activity list
 * @param {Array} activities - Array of activity objects
 */
function updateActivityList(activities) {
    if (!Array.isArray(activities) || activities.length === 0) {
        activityList.innerHTML = '<li class="activity-item">No recent activity</li>';
        return;
    }

    activityList.innerHTML = activities.map(activity => `
        <li class="activity-item">${escapeHtml(activity.message)} - ${escapeHtml(activity.timestamp)}</li>
    `).join('');
}

/**
 * Initialize event listeners
 */
function initEventListeners() {
    // Navigation
    navLinks.forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const sectionId = link.dataset.section;
            navigateTo(sectionId);
        });
    });

    // Sensitivity slider
    sensitivitySlider.addEventListener('input', () => {
        sensitivityValue.textContent = sensitivitySlider.value;
    });

    // Save settings button
    saveSettingsBtn.addEventListener('click', saveSettings);

    // Add gesture button
    addGestureBtn.addEventListener('click', showAddGestureModal);
}

/**
 * Initialize application
 */
function init() {
    initEventListeners();

    // Initial data load
    checkHealth();
    loadGestures();

    // Periodic health check (every 30 seconds)
    setInterval(checkHealth, 30000);
}

// Start application when DOM is ready
document.addEventListener('DOMContentLoaded', init);
