// KeyViewer - Frontend Application

let currentLayout = null;
let currentKeymap = null;
let currentLayerIndex = 0;

const KEY_SIZE = 54; // Base key size in pixels
const KEY_GAP = 4;   // Gap between keys

// DOM elements
const layoutFile = document.getElementById('layout-file');
const layoutSelect = document.getElementById('layout-select');
const keymapFile = document.getElementById('keymap-file');
const keymapSelect = document.getElementById('keymap-select');
const layerTabs = document.getElementById('layer-tabs');
const keyboardContainer = document.getElementById('keyboard-container');
const statusMessage = document.getElementById('status-message');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadLayoutList();
    loadKeymapList();
    setupEventListeners();
});

function setupEventListeners() {
    layoutFile.addEventListener('change', handleLayoutUpload);
    layoutSelect.addEventListener('change', handleLayoutSelect);
    keymapFile.addEventListener('change', handleKeymapUpload);
    keymapSelect.addEventListener('change', handleKeymapSelect);
}

function setStatus(msg, isError = false) {
    statusMessage.textContent = msg;
    statusMessage.className = isError ? 'error' : '';
}

// Layout functions
async function loadLayoutList() {
    try {
        const response = await fetch('/api/layouts');
        const layouts = await response.json();

        layoutSelect.innerHTML = '<option value="">-- Select layout --</option>';
        (layouts || []).forEach(name => {
            const option = document.createElement('option');
            option.value = name;
            option.textContent = name;
            layoutSelect.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load layouts:', error);
    }
}

async function handleLayoutUpload(event) {
    const file = event.target.files[0];
    if (!file) return;

    setStatus('Uploading layout...');

    const formData = new FormData();
    formData.append('layout', file);

    try {
        const response = await fetch('/api/layout', {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            throw new Error(await response.text());
        }

        currentLayout = await response.json();
        setStatus(`Layout "${currentLayout.name}" uploaded (${currentLayout.keys.length} keys)`);

        await loadLayoutList();
        layoutSelect.value = currentLayout.name;

        renderKeyboard();
    } catch (error) {
        setStatus('Layout error: ' + error.message, true);
        console.error('Layout upload failed:', error);
    }
}

async function handleLayoutSelect(event) {
    const name = event.target.value;
    if (!name) {
        currentLayout = null;
        renderKeyboard();
        return;
    }

    try {
        const response = await fetch(`/api/layout/${name}`);
        if (!response.ok) throw new Error('Failed to load layout');

        currentLayout = await response.json();
        setStatus(`Layout "${name}" loaded (${currentLayout.keys.length} keys)`);
        renderKeyboard();
    } catch (error) {
        setStatus('Failed to load layout', true);
        console.error('Failed to load layout:', error);
    }
}

// Keymap functions
async function loadKeymapList() {
    try {
        const response = await fetch('/api/keymaps');
        const keymaps = await response.json();

        keymapSelect.innerHTML = '<option value="">-- Select keymap --</option>';
        (keymaps || []).forEach(name => {
            const option = document.createElement('option');
            option.value = name;
            option.textContent = name;
            keymapSelect.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load keymaps:', error);
    }
}

async function handleKeymapUpload(event) {
    const file = event.target.files[0];
    if (!file) return;

    setStatus('Uploading keymap...');

    const formData = new FormData();
    formData.append('keymap', file);

    try {
        const response = await fetch('/api/keymap', {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            throw new Error(await response.text());
        }

        currentKeymap = await response.json();
        currentLayerIndex = 0;

        const keyCount = currentKeymap.layers[0]?.keys?.length || 0;
        setStatus(`Keymap "${currentKeymap.name}" uploaded (${currentKeymap.layers.length} layers, ${keyCount} keys)`);

        await loadKeymapList();
        keymapSelect.value = currentKeymap.name;

        renderKeyboard();
    } catch (error) {
        setStatus('Keymap error: ' + error.message, true);
        console.error('Keymap upload failed:', error);
    }
}

async function handleKeymapSelect(event) {
    const name = event.target.value;
    if (!name) {
        currentKeymap = null;
        layerTabs.innerHTML = '';
        renderKeyboard();
        return;
    }

    try {
        const response = await fetch(`/api/keymap/${name}`);
        if (!response.ok) throw new Error('Failed to load keymap');

        currentKeymap = await response.json();
        currentLayerIndex = 0;
        setStatus(`Keymap "${name}" loaded`);
        renderKeyboard();
    } catch (error) {
        setStatus('Failed to load keymap', true);
        console.error('Failed to load keymap:', error);
    }
}

// Rendering
function renderKeyboard() {
    if (!currentLayout) {
        keyboardContainer.innerHTML = '<p class="placeholder">Upload a KLE layout JSON to visualize the keyboard</p>';
        layerTabs.innerHTML = '';
        return;
    }

    if (currentKeymap) {
        renderLayerTabs();
    } else {
        layerTabs.innerHTML = '';
    }

    const keyboard = document.createElement('div');
    keyboard.className = 'keyboard';

    // Calculate bounds for centering
    let minX = Infinity, minY = Infinity, maxX = 0, maxY = 0;
    currentLayout.keys.forEach(key => {
        minX = Math.min(minX, key.x);
        minY = Math.min(minY, key.y);
        maxX = Math.max(maxX, key.x + key.w);
        maxY = Math.max(maxY, key.y + key.h);
    });

    const width = (maxX - minX) * KEY_SIZE + KEY_GAP;
    const height = (maxY - minY) * KEY_SIZE + KEY_GAP;
    keyboard.style.width = width + 'px';
    keyboard.style.height = height + 'px';

    // Get current layer keys
    const layerKeys = currentKeymap?.layers[currentLayerIndex]?.keys || [];

    // Render each key
    currentLayout.keys.forEach((physKey, index) => {
        const keyEl = document.createElement('div');
        const label = layerKeys[index] || '';

        keyEl.className = 'key ' + getKeyClass(label);
        keyEl.textContent = label;
        keyEl.title = `Key ${index}: ${label || 'empty'}`;

        // Position and size
        const x = (physKey.x - minX) * KEY_SIZE;
        const y = (physKey.y - minY) * KEY_SIZE;
        const w = physKey.w * KEY_SIZE - KEY_GAP;
        const h = physKey.h * KEY_SIZE - KEY_GAP;

        keyEl.style.left = x + 'px';
        keyEl.style.top = y + 'px';
        keyEl.style.width = w + 'px';
        keyEl.style.height = h + 'px';

        // Rotation
        if (physKey.r !== 0) {
            const rx = (physKey.rx - minX) * KEY_SIZE;
            const ry = (physKey.ry - minY) * KEY_SIZE;
            keyEl.style.transformOrigin = `${rx - x}px ${ry - y}px`;
            keyEl.style.transform = `rotate(${physKey.r}deg)`;
        }

        keyboard.appendChild(keyEl);
    });

    keyboardContainer.innerHTML = '';
    keyboardContainer.appendChild(keyboard);
}

function renderLayerTabs() {
    layerTabs.innerHTML = '';

    currentKeymap.layers.forEach((layer, index) => {
        const tab = document.createElement('button');
        tab.className = 'layer-tab' + (index === currentLayerIndex ? ' active' : '');
        tab.textContent = layer.name;
        tab.addEventListener('click', () => {
            currentLayerIndex = index;
            renderKeyboard();
        });
        layerTabs.appendChild(tab);
    });
}

function getKeyClass(key) {
    if (!key || key === '') return 'empty';
    if (key === '▽') return 'trans';
    if (key.startsWith('[') && key.endsWith(']')) return 'layer';
    if (key.includes('/')) return 'layer';
    if (['CTRL', 'ALT', 'GUI', 'SHFT', 'SHIFT'].some(m => key.includes(m))) return 'mod';
    if (['BOOT', 'BT', 'BL', 'LDR', 'CAPS', 'STUDIO'].some(s => key.startsWith(s))) return 'special';
    return '';
}
