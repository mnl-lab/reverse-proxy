from flask import Flask
import time

app = Flask(__name__)


@app.route("/")
def home():
    # Simulate heavy DB query
    time.sleep(2)
    return {"service": "Python Analytics API", "status": "success", "port": 8082}


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8082)
