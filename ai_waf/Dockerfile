# Use a lightweight Python base image
FROM python:3.9-slim

# Set the working directory inside the container
WORKDIR /app

# Copy the requirements file first (for caching layers)
COPY requirements.txt .

# Install dependencies
# --no-cache-dir keeps the image small
RUN pip install --no-cache-dir -r requirements.txt

# Copy the rest of the application code
# This includes app.py AND your trained waf_model.pkl
COPY . .

# Expose port 5000 for the Go Proxy
EXPOSE 5000

# Run the Flask application
CMD ["python", "app.py"]