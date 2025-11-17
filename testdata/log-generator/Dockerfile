FROM python:3.11-slim

WORKDIR /app

# Install dependencies
RUN pip install --no-cache-dir requests

# Copy generator script
COPY generate.py .

# Run generator
CMD ["python", "generate.py"]
