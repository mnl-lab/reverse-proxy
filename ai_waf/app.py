from flask import Flask, request, jsonify
from urllib.parse import unquote_plus
import pickle
import pandas as pd
import os
import re

app = Flask(__name__)

# Load trained model at startup
print("Loading AI Security Model...")
model_path = os.path.join(os.path.dirname(__file__), "waf_model.pkl")

try:
    with open(model_path, "rb") as f:
        vectorizer, model = pickle.load(f)
    print("Model loaded successfully!")
except FileNotFoundError:
    print("ERROR: Model file not found. Did you run train.py?")
    vectorizer, model = None, None


# Prediction endpoint
@app.route("/predict", methods=["POST"])
def predict():
    if not model:
        return jsonify({"error": "Model not loaded"}), 500

    # Parse JSON body
    data = request.get_json()

    url_to_check = data.get("url", "")

    if not url_to_check:
        return jsonify({"error": "No URL provided"}), 400

    # Decode percent-encoded characters
    decoded_url = unquote_plus(url_to_check)

    # Normalize digits so unseen values still align with training data
    cleaned_url = re.sub(r"\d", "0", decoded_url)

    print(f"Analyzing: {cleaned_url}")

    # Vectorize and score the URL
    vectorized_url = vectorizer.transform([cleaned_url])
    prediction = model.predict(vectorized_url)[0]

    # Map numeric label to human-readable verdict
    result = "SAFE" if prediction == 0 else "MALICIOUS"

    return jsonify(
        {"url": url_to_check, "prediction": int(prediction), "status": result}
    )


if __name__ == "__main__":
    # Serve Flask app on port 5000
    app.run(host="0.0.0.0", port=5000)
