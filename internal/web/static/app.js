let currentRoomId = null;
let topology = { groups: [] };
let ws = null;

async function init() {
    await fetchTopology();
    setupWebSocket();
    setupEventListeners();
}

async function fetchTopology() {
    try {
        const response = await fetch('/api/topology');
        topology = await response.json();
        renderRoomList();
        
        // Auto-select first room if none selected
        if (!currentRoomId && topology.groups && topology.groups.length > 0) {
            selectRoom(topology.groups[0].id);
        }
    } catch (err) {
        console.error('Failed to fetch topology:', err);
    }
}

function setupWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws`);
    
    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (data.type === 'topology') {
            topology = data.payload;
            renderRoomList();
        } else if (data.type === 'room_update') {
            if (data.roomId === currentRoomId) {
                updateRoomUI(data.payload);
            }
        } else if (data.type === 'refresh_hint') {
            if (currentRoomId) {
                fetchRoomStatus(currentRoomId);
            }
        }
    };
    
    ws.onclose = () => {
        console.log('WebSocket closed. Retrying in 5s...');
        setTimeout(setupWebSocket, 5000);
    };
}

function renderRoomList() {
    const roomList = document.getElementById('room-list');
    roomList.innerHTML = '';
    
    topology.groups.forEach(group => {
        const item = document.createElement('div');
        item.className = 'room-item';
        if (group.id === currentRoomId) item.classList.add('active');
        
        const memberNames = group.members.map(m => m.name).join(', ');
        
        item.innerHTML = `
            <span class="group-label">${group.coordinator.name}</span>
            <span class="member-names">${memberNames}</span>
        `;
        
        item.onclick = () => selectRoom(group.id);
        roomList.appendChild(item);
    });
}

async function selectRoom(roomId) {
    currentRoomId = roomId;
    renderRoomList();
    
    document.getElementById('no-room-selected').style.display = 'none';
    document.getElementById('room-view').style.display = 'block';
    
    await fetchRoomStatus(roomId);
    fetchQueue(roomId);
    fetchFavorites(roomId);
    fetchRoomInputs(roomId);
    renderGroupingUI();
}

function renderGroupingUI() {
    const list = document.getElementById('group-members-list');
    list.innerHTML = '';
    
    topology.groups.forEach(group => {
        // Check if current room is in this group
        let isSelf = false;
        group.members.forEach(m => {
            if (m.uuid === currentRoomId) isSelf = true;
        });
        
        if (!isSelf) {
            const btn = document.createElement('button');
            btn.textContent = `Join ${group.coordinator.name}`;
            btn.onclick = () => sendControl('joingroup', { meta: group.coordinator.uuid });
            list.appendChild(btn);
        }
    });
    
    const leaveBtn = document.createElement('button');
    leaveBtn.textContent = 'Leave Group (Solo)';
    leaveBtn.onclick = () => sendControl('leavegroup');
    list.appendChild(document.createElement('hr'));
    list.appendChild(leaveBtn);
}

async function fetchQueue(roomId) {
    try {
        const response = await fetch(`/api/room/${roomId}/queue`);
        const items = await response.json();
        renderList('queue-list', items, (item) => sendControl('playtrack', { value: item.id }));
    } catch (err) {
        console.error('Failed to fetch queue:', err);
    }
}

async function fetchFavorites(roomId) {
    try {
        const response = await fetch(`/api/room/${roomId}/favorites`);
        const items = await response.json();
        renderList('favorites-list', items, (item) => sendControl('playuri', { uri: item.uri, meta: item.resMD }));
    } catch (err) {
        console.error('Failed to fetch favorites:', err);
    }
}

function renderList(elementId, items, onPlay) {
    const list = document.getElementById(elementId);
    list.innerHTML = '';
    if (!items || items.length === 0) {
        list.innerHTML = '<div class="loading">No items found</div>';
        return;
    }
    
    items.forEach(item => {
        const div = document.createElement('div');
        div.className = 'list-item';
        div.innerHTML = `
            <div class="item-info">
                <span class="item-title">${item.title}</span>
                <span class="item-subtitle">${item.artist || ''}</span>
            </div>
            <button class="play-item-btn">▶</button>
        `;
        div.querySelector('.play-item-btn').onclick = () => onPlay(item);
        list.appendChild(div);
    });
}

async function fetchRoomStatus(roomId) {
    try {
        const response = await fetch(`/api/room/${roomId}/status`);
        const status = await response.json();
        updateRoomUI(status);
    } catch (err) {
        console.error('Failed to fetch room status:', err);
    }
}

async function fetchRoomInputs(roomId) {
    try {
        const response = await fetch(`/api/room/${roomId}/inputs`);
        const inputs = await response.json();
        renderInputsList(inputs);
    } catch (err) {
        console.error('Failed to fetch room inputs:', err);
    }
}

function renderInputsList(inputs) {
    const list = document.getElementById('inputs-list');
    list.innerHTML = '';
    
    if (inputs.length === 0) {
        list.innerHTML = '<div class="empty-list">No physical inputs found for this room.</div>';
        return;
    }
    
    inputs.forEach(input => {
        const item = document.createElement('div');
        item.className = 'list-item';
        item.innerHTML = `
            <div class="item-info">
                <div class="item-title">${input.title}</div>
            </div>
            <button class="play-item-btn">Switch</button>
        `;
        item.onclick = () => sendControl('playuri', { uri: input.uri, meta: input.meta || '' });
        list.appendChild(item);
    });
}

function updateRoomUI(status) {
    console.log('Updating Room UI', status);
    document.getElementById('current-room-name').textContent = status.name;
    document.getElementById('track-title').textContent = status.track || 'No Track';
    document.getElementById('track-artist').textContent = status.artist || '';
    document.getElementById('track-album').textContent = status.album || '';
    
    const albumArt = document.getElementById('album-art');
    if (status.albumArt) {
        albumArt.src = status.albumArt;
    } else {
        albumArt.src = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIiBzdHJva2U9IiM0NDQiIHN0cm9rZS13aWR0aD0iMiIgc3Ryb2tlLWxpbmVjYXA9InJvdW5kIiBzdHJva2UtbGluZWpvaW49InJvdW5kIj48cmVjdCB4PSIyIiB5PSIyIiB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHJ4PSI1IiByeT0iNSI+PC9yZWN0PjxjaXJjbGUgY3g9IjEyIiBjeT0iMTIiIHI9IjMiPjwvY2lyY2xlPjwvc3ZnPg==';
    }
    
    document.getElementById('play-pause-btn').textContent = status.state === 'PLAYING' ? '⏸' : '▶';
    document.getElementById('volume-slider').value = status.volume;
    
    // Progress
    const progress = document.getElementById('progress-bar');
    progress.max = status.durationSeconds || 100;
    progress.value = status.positionSeconds || 0;
    
    document.getElementById('current-time').textContent = formatTime(status.positionSeconds);
    document.getElementById('total-time').textContent = formatTime(status.durationSeconds);
    
    // EQ
    document.getElementById('bass-slider').value = status.bass;
    document.getElementById('treble-slider').value = status.treble;
    
    console.log('Setting mode indicators:', {
        loudness: status.loudness,
        nightMode: status.nightMode,
        speech: status.speechEnhancement
    });

    document.getElementById('loudness-btn').classList.toggle('active', !!status.loudness);
    document.getElementById('night-btn').classList.toggle('active', !!status.nightMode);
    document.getElementById('speech-btn').classList.toggle('active', !!status.speechEnhancement);
    
    // Grouping
    document.getElementById('group-volume-slider').value = status.groupVolume;
    document.getElementById('group-mute-btn').classList.toggle('active', status.groupMute);
}

function setupEventListeners() {
    document.getElementById('play-pause-btn').onclick = () => sendControl('playpause');
    document.getElementById('prev-btn').onclick = () => sendControl('previous');
    document.getElementById('next-btn').onclick = () => sendControl('next');
    
    document.getElementById('volume-slider').oninput = (e) => sendControl('volume', { value: parseInt(e.target.value) });
    document.getElementById('bass-slider').oninput = (e) => sendControl('bass', { value: parseInt(e.target.value) });
    document.getElementById('treble-slider').oninput = (e) => sendControl('treble', { value: parseInt(e.target.value) });
    
    document.getElementById('loudness-btn').onclick = () => sendControl('loudness');
    document.getElementById('night-btn').onclick = () => sendControl('nightmode');
    document.getElementById('speech-btn').onclick = () => sendControl('speechenhancement');
    
    document.getElementById('group-volume-slider').oninput = (e) => sendControl('groupvolume', { value: parseInt(e.target.value) });
    document.getElementById('group-mute-btn').onclick = () => sendControl('groupmute');
    
    // Progress seek
    document.getElementById('progress-bar').onchange = (e) => sendControl('seek', { value: parseInt(e.target.value) });
}

async function sendControl(command, params = {}) {
    if (!currentRoomId) {
        console.warn('No room selected for control');
        return;
    }
    console.log(`Sending control: ${command}`, params, `to room: ${currentRoomId}`);
    try {
        const response = await fetch(`/api/room/${currentRoomId}/control`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ command, ...params })
        });
        if (!response.ok) {
            console.error(`Control failed with status: ${response.status}`);
        }
    } catch (err) {
        console.error('Failed to send control:', err);
    }
}

window.onload = init;
