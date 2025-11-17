FROM python:3.11-slim

WORKDIR /app

# Install dependencies
RUN pip install --no-cache-dir flask requests

# Copy proxy script
COPY proxy.py .

# Expose proxy port
EXPOSE 8080

# Run proxy
CMD ["python", "proxy.py"]
