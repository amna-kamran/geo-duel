const statusEl = document.getElementById('status');
const placeEl = document.getElementById('place');
const submitBtn = document.getElementById('submitBtn');
const nextBtn = document.getElementById('nextBtn');
const resultEl = document.getElementById('result');
const yourTotalEl = document.getElementById('yourTotal');
const oppTotalEl = document.getElementById('oppTotal');

const map = L.map('map').setView([20, 0], 2);
L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
  attribution: '&copy; OpenStreetMap contributors',
}).addTo(map);

let guessMarker = null;
let currentGuess = null;
let roundLayers = [];
let canGuess = false;

function clearRoundLayers() {
  roundLayers.forEach((layer) => map.removeLayer(layer));
  roundLayers = [];
  if (guessMarker) {
    map.removeLayer(guessMarker);
    guessMarker = null;
  }
  currentGuess = null;
}

map.on('click', (e) => {
  if (!canGuess) return;
  currentGuess = e.latlng;
  if (guessMarker) map.removeLayer(guessMarker);
  guessMarker = L.marker(e.latlng).addTo(map);
  submitBtn.disabled = false;
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

const proto = location.protocol === 'https:' ? 'wss' : 'ws';
const ws = new WebSocket(`${proto}://${location.host}/ws`);

ws.onopen = () => {
  statusEl.textContent = 'Connected';
};

ws.onclose = () => {
  statusEl.textContent = 'Disconnected';
};

ws.onerror = () => {
  statusEl.textContent = 'Connection error';
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);

  switch (msg.type) {
    case 'waiting':
      statusEl.textContent = 'Waiting for opponent to join...';
      break;

    case 'round_start':
      statusEl.textContent = 'Your turn — place your guess';
      placeEl.textContent = `Find: ${msg.place}`;
      resultEl.textContent = '';
      nextBtn.style.display = 'none';
      submitBtn.style.display = 'inline-block';
      submitBtn.disabled = true;
      canGuess = true;
      clearRoundLayers();
      break;

    case 'round_result': {
      canGuess = false;
      submitBtn.style.display = 'none';
      nextBtn.style.display = 'inline-block';
      nextBtn.disabled = false;
      statusEl.textContent = 'Round complete';

      const actual = L.marker([msg.actual.lat, msg.actual.lng])
        .addTo(map)
        .bindPopup('Actual location')
        .openPopup();
      roundLayers.push(actual);

      const you = L.circleMarker([msg.yourGuess.lat, msg.yourGuess.lng], { color: '#2563eb' })
        .addTo(map)
        .bindPopup(`Your guess (+${msg.yourScore})`);
      roundLayers.push(you);

      const opp = L.circleMarker([msg.opponentGuess.lat, msg.opponentGuess.lng], { color: '#dc2626' })
        .addTo(map)
        .bindPopup(`Opponent guess (+${msg.opponentScore})`);
      roundLayers.push(opp);

      const line1 = L.polyline(
        [[msg.yourGuess.lat, msg.yourGuess.lng], [msg.actual.lat, msg.actual.lng]],
        { color: '#2563eb', dashArray: '4' }
      ).addTo(map);
      const line2 = L.polyline(
        [[msg.opponentGuess.lat, msg.opponentGuess.lng], [msg.actual.lat, msg.actual.lng]],
        { color: '#dc2626', dashArray: '4' }
      ).addTo(map);
      roundLayers.push(line1, line2);

      yourTotalEl.textContent = msg.yourTotal;
      oppTotalEl.textContent = msg.opponentTotal;

      resultEl.textContent = `You scored ${msg.yourScore}, opponent scored ${msg.opponentScore}.`;
      break;
    }
  }
};
