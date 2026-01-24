const http = require('http');

const server = http.createServer((req, res) => {
    // Simulate image processing delay
    setTimeout(() => {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ service: "Node.js Render Engine", status: "success", port: 8083 }));
    }, 2000); 
});

server.listen(8083, '0.0.0.0', () => {
    console.log('Node backend running on port 8083');
});