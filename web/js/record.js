// web/js/record.js - Gesture Recording UI

const API_BASE = '/api';
let ws = null;
let recording = false;
let samples = [];
let currentLandmarks = null;
let pathBuffer = [];

// DOM elements
const video = document.getElementById('camera');
const canvas = document.getElementById('overlay');
const ctx = canvas.getContext('2d');
const gestureNameInput = document.getElementById('gesture-name');
const gestureTypeSelect = document.getElementById('gesture-type');
const sampleList = document.getElementById('sample-list');
const sampleCount = document.getElementById('sample-count');
const recordBtn = document.getElementById('record-btn');
const saveBtn = document.getElementById('save-btn');

// Hand landmark connections for drawing
const HAND_CONNECTIONS = [
    [0, 1], [1, 2], [2, 3], [3, 4],           // Thumb
    [0, 5], [5, 6], [6, 7], [7, 8],           // Index
    [0, 9], [9, 10], [10, 11], [11, 12],      // Middle
    [0, 13], [13, 14], [14, 15], [15, 16],    // Ring
    [0, 17], [17, 18], [18, 19], [19, 20],    // Pinky
    [5, 9], [9, 13], [13, 17]                 // Palm
];

// Initialize camera using WebRTC
async function initCamera() {
    try {
        const stream = await navigator.mediaDevices.getUserMedia({
            video: { width: 640, height: 480 }
        });
        video.srcObject = stream;
        video.onloadedmetadata = () => {
            canvas.width = video.videoWidth;
            canvas.height = video.videoHeight;
            recordBtn.disabled = false;
        };
    } catch (err) {
        console.error('Camera error:', err);
        alert('Could not access camera. Please grant permission.');
    }
}

// Connect to landmarks WebSocket
function connectWebSocket() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${location.host}/api/landmarks`);

    ws.onopen = () => {
        console.log('WebSocket connected');
    };

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        currentLandmarks = data.hands;
        drawLandmarks(data.hands);

        if (recording && gestureTypeSelect.value === 'dynamic') {
            // Buffer path points for dynamic gestures (use wrist position)
            if (data.hands && data.hands.length > 0) {
                const points = data.hands[0].points;
                if (points && points.length > 0) {
                    pathBuffer.push({
                        x: points[0].x,  // Wrist X
                        y: points[0].y,  // Wrist Y
                        timestamp: data.timestamp
                    });
                }
            }
        }
    };

    ws.onerror = (err) => {
        console.error('WebSocket error:', err);
    };

    ws.onclose = () => {
        console.log('WebSocket closed, reconnecting...');
        setTimeout(connectWebSocket, 1000);
    };
}

// Draw hand landmarks on canvas
function drawLandmarks(hands) {
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    if (!hands || hands.length === 0) {
        return;
    }

    for (const hand of hands) {
        const points = hand.points;
        if (!points || points.length < 21) {
            continue;
        }

        // Draw connections
        ctx.strokeStyle = '#e94560';
        ctx.lineWidth = 2;

        for (const [i, j] of HAND_CONNECTIONS) {
            const p1 = points[i];
            const p2 = points[j];
            ctx.beginPath();
            ctx.moveTo(p1.x * canvas.width, p1.y * canvas.height);
            ctx.lineTo(p2.x * canvas.width, p2.y * canvas.height);
            ctx.stroke();
        }

        // Draw points
        ctx.fillStyle = '#fff';
        for (const point of points) {
            ctx.beginPath();
            ctx.arc(point.x * canvas.width, point.y * canvas.height, 4, 0, Math.PI * 2);
            ctx.fill();
        }
    }
}

// Record a sample
function recordSample() {
    const type = gestureTypeSelect.value;

    if (type === 'static') {
        // Capture current landmarks
        if (!currentLandmarks || currentLandmarks.length === 0) {
            alert('No hand detected. Please position your hand in view.');
            return;
        }

        samples.push({
            type: 'static',
            landmarks: currentLandmarks[0].points,
            timestamp: Date.now()
        });

        updateSampleList();
    } else {
        // Record dynamic gesture (2 seconds)
        recording = true;
        pathBuffer = [];
        recordBtn.textContent = 'Recording...';
        recordBtn.classList.add('recording');
        recordBtn.disabled = true;

        setTimeout(() => {
            recording = false;
            recordBtn.textContent = 'Record Sample';
            recordBtn.classList.remove('recording');
            recordBtn.disabled = false;

            if (pathBuffer.length < 10) {
                alert('Not enough movement detected. Please try again.');
                return;
            }

            samples.push({
                type: 'dynamic',
                path: pathBuffer,
                timestamp: Date.now()
            });

            updateSampleList();
        }, 2000);
    }
}

// Update sample list UI
function updateSampleList() {
    const count = samples.length;
    sampleCount.textContent = count;

    if (count === 0) {
        sampleList.innerHTML = '<p class="text-muted">No samples recorded yet</p>';
    } else {
        sampleList.innerHTML = samples.map((s, i) => `
            <div class="sample-item">
                <span>Sample ${i + 1} (${s.type})</span>
                <button onclick="removeSample(${i})" title="Remove sample">&times;</button>
            </div>
        `).join('');
    }

    saveBtn.disabled = count < 3;
}

// Remove a sample
function removeSample(index) {
    samples.splice(index, 1);
    updateSampleList();
}

// Save gesture
async function saveGesture() {
    const name = gestureNameInput.value.trim();
    if (!name) {
        alert('Please enter a gesture name.');
        return;
    }

    const type = gestureTypeSelect.value;

    try {
        saveBtn.disabled = true;
        saveBtn.textContent = 'Saving...';

        // Create gesture
        const response = await fetch(`${API_BASE}/gestures`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, type })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to create gesture');
        }

        const gesture = await response.json();

        // Save samples
        const samplesResponse = await fetch(`${API_BASE}/gestures/${gesture.id}/samples`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ samples })
        });

        if (!samplesResponse.ok) {
            throw new Error('Failed to save samples');
        }

        alert('Gesture saved successfully!');
        window.location.href = '/';
    } catch (err) {
        console.error('Save error:', err);
        alert('Failed to save gesture: ' + err.message);
        saveBtn.disabled = false;
        saveBtn.textContent = 'Save Gesture';
    }
}

// Type change handler
gestureTypeSelect.onchange = () => {
    const isStatic = gestureTypeSelect.value === 'static';
    document.getElementById('static-instructions').style.display = isStatic ? 'block' : 'none';
    document.getElementById('dynamic-instructions').style.display = isStatic ? 'none' : 'block';
    // Clear samples when changing type
    samples = [];
    updateSampleList();
};

// Event listeners
recordBtn.onclick = recordSample;
saveBtn.onclick = saveGesture;

// Make removeSample available globally for onclick handlers
window.removeSample = removeSample;

// Initialize
initCamera();
connectWebSocket();
