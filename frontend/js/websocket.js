let ws;

function connectWebSocket() {
  ws = new WebSocket('ws://localhost:8080/ws');  // Change to your Go server URL later

  ws.onopen = () => {
    console.log('WebSocket Connected');
  };

  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'new_request') {
      // Add to live requests feed
      addRequestToFeed(data);
    } else if (data.type === 'chat_message') {
      // Add to chat screen
      addMessageToChat(data);
    }
  };

  ws.onclose = () => {
    console.log('WebSocket Closed');
  };
}

function sendMessage(message) {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(message));
  }
}

// Example: Add request to feed (call in live-requests.html)
function addRequestToFeed(data) {
  const feed = document.getElementById('request-feed');
  if (feed) {
    const card = document.createElement('div');
    card.className = 'card';
    card.innerHTML = `
      <img src="${data.photo}" alt="User">
      <div class="card-content">
        <div class="card-title">${data.name}, ${data.age}</div>
        <div class="card-text">${data.text}</div>
      </div>
    `;
    feed.appendChild(card);
  }
}

// Example: Add message to chat
function addMessageToChat(data) {
  const chat = document.getElementById('chat-messages');
  if (chat) {
    const msg = document.createElement('div');
    msg.textContent = `${data.sender}: ${data.text}`;
    chat.appendChild(msg);
  }
}

// Connect on load
connectWebSocket();