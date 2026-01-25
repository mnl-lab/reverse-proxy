const express = require('express');
const app = express();

// Read environment variables passed by Docker
const SERVER_ID = process.env.SERVER_ID || "Unknown-Server";
const SERVER_COLOR = process.env.SERVER_COLOR || "#333333";
const PORT = process.env.PORT || 8080;

app.get('/', (req, res) => {
    const query = req.query.search || "";
    
    // Simulate database lookup vulnerability
    let dbResult = "Enter a search term.";
    if (query === "' OR 1=1 --") {
        dbResult = "<span style='color:red; font-weight:bold;'>CRITICAL BREACH: [Admin: p@ssword1], [User: 12345]</span>";
    } else if (query !== "") {
        dbResult = `Showing 0 results for: "<strong>${query}</strong>"`;
    }

    // Modern HTML Dashboard
    const html = `
        <!DOCTYPE html>
        <html>
        <head>
            <title>MNL Dashboard</title>
            <style>
                body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #f4f7f6; margin: 0; }
                .header { background: ${SERVER_COLOR}; color: white; padding: 20px; text-align: center; font-size: 24px; font-weight: bold; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
                .container { max-width: 800px; margin: 40px auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 0 15px rgba(0,0,0,0.05); }
                .search-bar { width: 70%; padding: 12px; font-size: 16px; border: 1px solid #ccc; border-radius: 4px; }
                .btn { padding: 12px 24px; font-size: 16px; background: ${SERVER_COLOR}; color: white; border: none; border-radius: 4px; cursor: pointer; }
                .console { background: #1e1e1e; color: #00ff00; padding: 20px; border-radius: 4px; font-family: monospace; margin-top: 20px; }
            </style>
        </head>
        <body>
            <div class="header">
                [ Connected to: ${SERVER_ID} ]
            </div>
            <div class="container">
                <h2>Product Search Engine</h2>
                <form method="GET" action="/">
                    <input type="text" name="search" class="search-bar" placeholder="Search product name or SKU..." value="${query}">
                    <button type="submit" class="btn">Search</button>
                </form>
                
                <div class="console">
                    > System Query Log:<br>
                    > ${dbResult}
                </div>
            </div>
        </body>
        </html>
    `;
    res.send(html);
});

app.listen(PORT, '0.0.0.0', () => {
    console.log(`${SERVER_ID} running on port ${PORT}`);
});