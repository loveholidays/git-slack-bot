FROM europe-docker.pkg.dev/loveholidays-ci-cd/external-images-virtual/python:3.13-slim@sha256:fea139ecc0ab98360ead50b38872df6d42cf3503bec83215461224cbef0c8269
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY remind.py test_remind.py ./
RUN python -m unittest discover -p 'test_*.py'
USER 65532:65532
ENTRYPOINT ["python", "/app/remind.py"]
