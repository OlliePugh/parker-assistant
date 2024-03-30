import net from "net";

const socketPath = "/tmp/plugin_socket";

const client = new net.Socket();

console.log("whats up brother");

client.connect(socketPath, () => {
  console.log("Connected to master application");

  // Send message to master application
  const message = "music";
  client.write(message);

  // Receive response from master application
  client.on("data", (data) => {
    console.log("Received response from master:", data.toString());
    client.end();
  });

  // Handle socket closure
  client.on("close", () => {
    console.log("Connection closed");
  });
});

// Handle connection errors
client.on("error", (error) => {
  console.error("Error connecting to master application:", error);
});
