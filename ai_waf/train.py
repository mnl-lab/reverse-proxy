import pickle
from pathlib import Path

import pandas as pd
import re  # Regex used to normalize digits
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.linear_model import LogisticRegression
from sklearn.pipeline import Pipeline  # Provides preprocessing composition helpers

print("1. Loading Advanced Dataset...")


# Normalize digits so numeric variants share a single representation
def clean_url(url):
    # Replace digits with a neutral token
    return re.sub(r"\d", "0", url)

data = {
    "url": [
        # --- SAFE (Normal User Behavior) ---
        # We MUST include the exact patterns our app uses (/?search=)
        "/?search=shoes",
        "/?search=product",
        "/?search=iphone",
        "/?search=black+boots",
        "/?search=category=men",
        "/home",
        "/login",
        "/dashboard",
        "/contact-us",
        "/about",
        "/api/v1/users",
        "/profile?user=john",
        "/cart/add?item=55",
        # --- MALICIOUS (SQL Injection) ---
        "/?search=' OR 1=1 --",
        "/?search=' OR 2=2 --",  
        "/?search=' OR 0=0 #",
        "/login?user=admin' #",
        "/products?id=1 UNION SELECT username, password FROM users",
        "/dashboard?id=1; DROP TABLE users",
        "/?q=admin' AND 1=1",
        "/?search=' OR 100=100",
        "1=1",
        "' OR 'a'='a'",
        # --- MALICIOUS (XSS) ---
        "/search?q=<script>alert('hacked')</script>",
        "/profile?name=<img src=x onerror=alert(1)>",
        "/?ref=javascript:alert(document.cookie)",
    ],
    "label": [
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        0,
        0,  # 13 Safe examples
        1,
        1,
        1,
        1,
        1,
        1,
        1,
        1,
        1,
        1,  # 10 SQLi
        1,
        1,
        1,  # 3 XSS
    ],
}
df = pd.DataFrame(data)

# Normalize URLs before feature extraction
df["clean_url"] = df["url"].apply(clean_url)

print("2. Vectorizing URLs (with Number Normalization)...")
# Character n-grams capture structure after digit normalization
vectorizer = TfidfVectorizer(analyzer="char", ngram_range=(2, 4))
X = vectorizer.fit_transform(df["clean_url"])
y = df["label"]

print("3. Training the Model...")
model = LogisticRegression()
model.fit(X, y)

print("4. Testing New Capabilities...")
# Basic regression tests for tricky cases
test_cases = [
    "/search?q=' OR 2=2",  # Numeric tautology
    "/search?q=' OR 500=500",  # Large-number tautology
    "/products?id=999",  # Typical numeric parameter
]
cleaned_tests = [clean_url(u) for u in test_cases]
test_vectors = vectorizer.transform(cleaned_tests)
predictions = model.predict(test_vectors)

for url, pred in zip(test_cases, predictions):
    status = "MALICIOUS" if pred == 1 else "SAFE"
    print(f"URL: {url} -> {status}")

print("5. Saving the Upgraded Brain...")
model_path = Path(__file__).resolve().parent / "waf_model.pkl"
with open(model_path, "wb") as f:
    pickle.dump((vectorizer, model), f)

print("SUCCESS: Advanced Model Saved.")
