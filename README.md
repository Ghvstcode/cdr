### Summary Of Changes/Development Notes

- HTTP+WebSocket Server with In-Memory PubSub: Implemented a robust HTTP+WebSocket server leveraging an in-memory, real-time PubSub system. This design facilitates bi-directional and event-driven communication, essential for modern, responsive applications.
- Standardized Message Structure: Adopted a uniform message format encapsulated in the Message struct, ensuring consistency in message parsing and handling. This is critical for the efficient operation of the PubSub system.
```go
 type Message struct {  
Topic   string `json:"topic"`  
Content string `json:"message"`  
 }
 ```
- Dynamic Topic Subscription: Developed a mechanism to evaluate if a WebSocket connection is already subscribed to a given topic. If not, the system dynamically subscribes the connection and then broadcasts the message to all subscribers of that topic.
- Chat Room Demo Application: Included a chat application as a practical demonstration of the server's capabilities, allowing users to create and join chat rooms. This feature can be tested by running the server and visiting localhost:8000.
- Use of Unbuffered Channels: Employed unbuffered channels for immediate message processing, assuming subscribers handle messages instantaneously. This approach is integral to the system's message queuing and processing strategy.
- Testing Challenges and Solutions: Addressed challenges in testing message delivery to multiple subscribers, especially in simulating subscription establishment and synchronization in test environments.
- Client-Controlled Channel Creation: Conformed to Go's conventions by allowing clients to create channels, providing them with the flexibility to determine suitable buffer sizes and accommodating varied consumption patterns.
- Concurrency Management with Channels: Utilized channels for concurrency control, adhering to Go's philosophy of memory sharing through communication. This strategy significantly enhances data integrity and synchronization.
- Omission of Unsubscribe Functionality: Noted the current absence of an unsubscribe feature in the PubSub system, acknowledging this as an area for future development to avoid potential resource leaks.
- Graceful Server Shutdown: Implemented a sophisticated server shutdown process using channels and system signals, ensuring a seamless termination of the PubSub service and the safe release of resources.
- WebSocket Ping-Pong Mechanism: Incorporated a ping-pong mechanism to continuously check client connectivity, playing a crucial role in maintaining active and responsive WebSocket connections.
- Integration of Gorilla WebSocket Library: Chose the Gorilla WebSocket library for upgrading HTTP connections to WebSocket, providing a standard and efficient method for WebSocket communication.
- SafeWebSocketConn for Enhanced Safety: Introduced the SafeWebSocketConn struct, equipped with a mutex to prevent concurrent write operations, thus boosting thread safety and connection stability.
- Thread-Safe PubSub Implementation: Developed the PubSub system with a focus on scalability and thread safety, using a read-write mutex to safeguard the subscription map during concurrent operations.
- Non-Blocking Message Broadcasts in PubSub: Designed the PubSub system to drop messages if a subscriber’s channel is not ready, prioritizing system responsiveness and avoiding blocking operations.
- Context-Based Server Management: Integrated Go's context package for effective lifecycle management of the server, particularly for handling system signals and shutdown procedures.
- Server Timeout Configurations: Configured server timeouts for read, write, and idle operations to mitigate resource exhaustion and enhance protection against potential DoS attacks.
- Clear WebSocket Message Handling Workflow: Structured the WebSocket message handler to initially establish the connection, followed by a continuous loop for reading and processing messages, ensuring clarity and maintainability in the workflow.
- Empowering Clients in Concurrency Control: Aligned with Go’s approach to concurrency by allowing client-driven channel creation, thus catering to diverse client needs and consumption behaviors.
- Decided to profile the application to get a bit more insights
<img width="965" alt="Screenshot 2023-11-18 at 10 35 51" src="https://github.com/Ghvstcode/cdr/assets/46195831/e4612e88-4e62-46a2-a4cc-2468e29ef5f8">
- This is a simple script I use to simulate multiple connections

```sh

#!/bin/bash

# Function to check if websocat is installed
check_websocat() {
    if ! command -v websocat &> /dev/null; then
        echo "websocat could not be found, attempting to install it..."

        # Detect OS and install websocat using the appropriate package manager
        case "$(uname -s)" in
            Linux*)
                if command -v apt-get &> /dev/null; then
                    sudo apt-get update && sudo apt-get install -y websocat
                elif command -v yum &> /dev/null; then
                    sudo yum install -y websocat
                # Add more package managers as needed
                else
                    echo "Package manager not supported. Please install websocat manually."
                    exit 1
                fi
                ;;
            Darwin*)
                if command -v brew &> /dev/null; then
                    brew install websocat
                else
                    echo "Homebrew not found. Please install websocat manually."
                    exit 1
                fi
                ;;
            *)
                echo "Operating system not supported. Please install websocat manually."
                exit 1
                ;;
        esac
    fi
}

# Check for websocat
check_websocat

numberOfClients=10000
serverUrl="ws://localhost:8000/ws"
messageInterval=1 # Interval in seconds

for ((i=0; i<numberOfClients; i++)); do
    (
        # Open a WebSocket connection
        websocat -E $serverUrl | while read -r line; do
            echo "Client $i received: $line"
        done &

        # Send messages at regular intervals
        while true; do
            echo "{\"topic\": \"test\", \"message\": \"Message from client $i\"}" | websocat $serverUrl
            sleep $messageInterval
        done
    ) &
    echo "Client $i connected"
done

wait
```
