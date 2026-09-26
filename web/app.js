const statusEl = document.getElementById('status');
const placeEl = document.getElementById('place');
const continentEl = document.getElementById('continent');
const opponentEl = document.getElementById('opponent');
const submitBtn = document.getElementById('submitBtn');
const nextBtn = document.getElementById('nextBtn');
const mapContainer = document.getElementById('map');
const resultEl = document.getElementById('result');
const yourTotalEl = document.getElementById('yourTotal');
const oppTotalEl = document.getElementById('oppTotal');
const leaderboardList = document.getElementById('leaderboardList');
const nameModal = document.getElementById('nameModal');
const nameForm = document.getElementById('nameForm');
const nameInput = document.getElementById('nameInput');

const DEFAULT_VIEW = { lat: 20, lng: 0, range: 8000000, tilt: 0 };
const REVEAL_VIEW = { range: 450000, tilt: 55 };

const ACCENT = { you: '#5b8dff', opponent: '#ff5b70', actual: '#ff0000' };
const NAME_KEY = 'geoduelName';

let map3d;
let Marker3DElement, Polyline3DElement, PinElement;
let guessMarker = null;
let roundEntities = [];
let currentGuess = null;
let canGuess = false;
let mapReady = null;
let ws = null;
let myName = '';

function makePin(color, glyph, scale = 1) {
  return new PinElement({
    background: color,
    borderColor: '#0b0b0d',
    glyphColor: '#ffffff',
    glyph,
    scale,
  });
}

function makeMarker(lat, lng, color, glyph, scale = 1) {
  const marker = new Marker3DElement({
    position: { lat, lng, altitude: 25 },
    altitudeMode: 'RELATIVE_TO_GROUND',
    collisionBehavior: 'REQUIRED_AND_HIDES_OPTIONAL',
  });
  marker.append(makePin(color, glyph, scale));
  map3d.append(marker);
  return marker;
}

function makeArc(from, to, color) {
  const line = new Polyline3DElement({
    coordinates: [
      { lat: from.lat, lng: from.lng, altitude: 15000 },
      { lat: to.lat, lng: to.lng, altitude: 15000 },
    ],
    strokeColor: color,
    strokeWidth: 3,
    altitudeMode: 'RELATIVE_TO_GROUND',
    geodesic: true,
  });
  map3d.append(line);
  return line;
}

function clearRoundLayers() {
  roundEntities.forEach((e) => e.remove());
  roundEntities = [];
  if (guessMarker) {
    guessMarker.remove();
    guessMarker = null;
  }
  currentGuess = null;
}

function setCameraView(lat, lng, range, tiltDeg, durationMs) {
  if (durationMs > 0) {
    map3d.flyCameraTo({
      endCamera: { center: { lat, lng, altitude: 0 }, range, tilt: tiltDeg, heading: 0 },
      durationMillis: durationMs,
    });
  } else {
    map3d.center = { lat, lng, altitude: 0 };
    map3d.range = range;
    map3d.tilt = tiltDeg;
    map3d.heading = 0;
  }
}

async function initMap() {
  const libs = await google.maps.importLibrary('maps3d');
  const markerLib = await google.maps.importLibrary('marker');
  ({ Marker3DElement, Polyline3DElement } = libs);
  ({ PinElement } = markerLib);

  map3d = new libs.Map3DElement({
    center: { lat: DEFAULT_VIEW.lat, lng: DEFAULT_VIEW.lng, altitude: 0 },
    range: DEFAULT_VIEW.range,
    tilt: DEFAULT_VIEW.tilt,
    heading: 0,
    mode: 'SATELLITE',
  });
  mapContainer.append(map3d);

  // Placing the guess pin never moves the camera, so there is nothing to
  // animate — the pin just appears where you click.
  map3d.addEventListener('gmp-click', (ev) => {
    if (!canGuess || !ev.position) return;
    const { lat, lng } = ev.position;
    currentGuess = { lat, lng };
    if (guessMarker) guessMarker.remove();
    guessMarker = makeMarker(lat, lng, ACCENT.actual, '?', 1.2);
    submitBtn.disabled = false;
  });
}

mapReady = initMap();
mapReady.catch((err) => {
  console.error(err);
  statusEl.textContent = 'Failed to load the 3D globe';
});

submitBtn.addEventListener('click', () => {
  if (!currentGuess) return;
  ws.send(JSON.stringify({ type: 'guess', lat: currentGuess.lat, lng: currentGuess.lng }));
  submitBtn.disabled = true;
  canGuess = false;
  statusEl.textContent = 'Waiting for opponent...';
});

nextBtn.addEventListener('click', () => {
  ws.send(JSON.stringify({ type: 'ready' }));
  nextBtn.disabled = true;
});

async function loadLeaderboard() {
  let list;
  try {
    const res = await fetch('/leaderboard');
    list = await res.json();
  } catch {
    return;
  }
  renderLeaderboard(list || []);
}

function renderLeaderboard(list) {
  leaderboardList.innerHTML = '';
  if (list.length === 0) {
    const li = document.createElement('li');
    li.className = 'empty';
    li.textContent = 'No rounds played yet — be the first on the board.';
    leaderboardList.append(li);
    return;
  }
  list.forEach((entry, i) => {
    const li = document.createElement('li');
    li.className = 'leaderboard-row' + (entry.name === myName ? ' me' : '');
    li.innerHTML = `
      <span class="leaderboard-rank">${i + 1}</span>
      <span class="leaderboard-name">${escapeHtml(entry.name)}</span>
      <span class="leaderboard-meta">${entry.wins}W / ${entry.games}G</span>
      <span class="leaderboard-score">${entry.totalScore}</span>
    `;
    leaderboardList.append(li);
  });
}

function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

function connect(name) {
  myName = name;
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  ws = new WebSocket(`${proto}://${location.host}/ws?name=${encodeURIComponent(name)}`);

  ws.onopen = () => {
    statusEl.textContent = 'Connected';
  };

  ws.onclose = () => {
    statusEl.textContent = 'Disconnected';
  };

  ws.onerror = () => {
    statusEl.textContent = 'Connection error';
  };

  ws.onmessage = async (event) => {
    const msg = JSON.parse(event.data);
    await mapReady;

    switch (msg.type) {
      case 'waiting':
        statusEl.textContent = 'Waiting for opponent to join...';
        opponentEl.textContent = '';
        opponentEl.classList.remove('vs-bot');
        break;

      case 'round_start':
        statusEl.textContent = 'Your turn — place your guess';
        placeEl.textContent = `Find: ${msg.place}`;
        continentEl.textContent = '';
        opponentEl.textContent = msg.isBot ? `Practicing vs ${msg.opponentName} 🤖` : `Playing vs ${msg.opponentName}`;
        opponentEl.classList.toggle('vs-bot', !!msg.isBot);
        resultEl.textContent = '';
        nextBtn.style.display = 'none';
        submitBtn.style.display = 'inline-block';
        submitBtn.disabled = true;
        canGuess = true;
        clearRoundLayers();
        map3d.mode = 'SATELLITE';
        setCameraView(DEFAULT_VIEW.lat, DEFAULT_VIEW.lng, DEFAULT_VIEW.range, DEFAULT_VIEW.tilt, 1500);
        break;

      case 'round_result': {
        canGuess = false;
        submitBtn.style.display = 'none';
        nextBtn.style.display = 'inline-block';
        nextBtn.disabled = false;
        statusEl.textContent = 'Round complete';
        continentEl.textContent = `📍 ${msg.continent}`;
        map3d.mode = 'HYBRID';

        const actualPos = { lat: msg.actual.lat, lng: msg.actual.lng };
        const yourPos = { lat: msg.yourGuess.lat, lng: msg.yourGuess.lng };
        const oppPos = { lat: msg.opponentGuess.lat, lng: msg.opponentGuess.lng };

        if (guessMarker) {
          guessMarker.remove();
          guessMarker = null;
        }

        roundEntities.push(
          makeMarker(actualPos.lat, actualPos.lng, ACCENT.actual, '★', 1.1),
          makeMarker(yourPos.lat, yourPos.lng, ACCENT.you, 'Y'),
          makeMarker(oppPos.lat, oppPos.lng, ACCENT.opponent, 'O'),
          makeArc(yourPos, actualPos, ACCENT.you),
          makeArc(oppPos, actualPos, ACCENT.opponent)
        );

        setCameraView(actualPos.lat, actualPos.lng, REVEAL_VIEW.range, REVEAL_VIEW.tilt, 0);

        yourTotalEl.textContent = msg.yourTotal;
        oppTotalEl.textContent = msg.opponentTotal;

        resultEl.textContent = `You scored ${msg.yourScore}, opponent scored ${msg.opponentScore}.`;
        loadLeaderboard();
        break;
      }
    }
  };
}

nameForm.addEventListener('submit', (e) => {
  e.preventDefault();
  const name = nameInput.value.trim();
  if (!name) return;
  localStorage.setItem(NAME_KEY, name);
  nameModal.classList.remove('open');
  connect(name);
});

loadLeaderboard();

const savedName = localStorage.getItem(NAME_KEY);
if (savedName) {
  connect(savedName);
} else {
  nameModal.classList.add('open');
  nameInput.focus();
}
