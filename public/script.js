let socket;
let currentRoom = '';

function generateRoomCode() {
    currentRoom = Math.random().toString(36).substring(2, 10); // Update currentRoom here
    return currentRoom;
}

function connectToRoom(room) {
    if (socket) {
        socket.close(); // Close existing connection if any
    }

    socket = new WebSocket('ws://localhost:8000/ws');

    socket.onopen = function() {
        // Send the room code to the server to subscribe to the room
        socket.send(JSON.stringify({ message: 'A new user just joined the chat', topic: room }));
        document.getElementById('messageInput').disabled = false;
        document.getElementById('sendMessageButton').disabled = false;
    };

    socket.onmessage = function(event) {
        //const data = JSON.parse(event.data);
        let messagesDiv = document.getElementById('messages');
        messagesDiv.innerHTML += `<p>${event.data}</p>`;
    };

    socket.onclose = function() {
        currentRoom = ''; // Reset currentRoom on disconnect
        updateCurrentRoomDisplay();
        document.getElementById('messageInput').disabled = true;
        document.getElementById('sendMessageButton').disabled = true;
    };
}

document.getElementById('joinRoomButton').onclick = function() {
    let roomInput = document.getElementById('roomInput');
    if (roomInput.value) {
        currentRoom = roomInput.value; // Update currentRoom here
        connectToRoom(currentRoom);
        updateCurrentRoomDisplay();
    }
};

document.getElementById('createRoomButton').onclick = function() {
    let newRoomCode = generateRoomCode(); // This updates currentRoom
    connectToRoom(newRoomCode);
    updateCurrentRoomDisplay();
};

document.getElementById('sendMessageButton').onclick = function() {
    let messageInput = document.getElementById('messageInput');
    if (messageInput.value) {
        // Include currentRoom as 'topic' in the JSON
        socket.send(JSON.stringify({ message: messageInput.value, topic: currentRoom }));
        messageInput.value = '';
    }
};

function updateCurrentRoomDisplay() {
    document.getElementById('currentRoom').innerText = currentRoom;
}
