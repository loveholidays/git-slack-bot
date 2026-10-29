FROM europe-docker.pkg.dev/loveholidays-ci-cd/external-images-virtual/python:3.13.18-slim@sha256:aaf08c5490d52cd92bc27142a5215b1e47355a5eaf9adfad2a0fbc022d0b5490
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY remind.py test_remind.py ./
RUN python -m unittest discover -p 'test_*.py'
USER 65532:65532
ENTRYPOINT ["python", "/app/remind.py"]
