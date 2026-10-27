FROM eu.gcr.io/loveholidays-ci-cd/python:3.13-slim@sha256:a1cb4989abf74fae1d8927cae2cff758f84b12586e701802bba3272b703899f3
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY remind.py test_remind.py ./
RUN python -m unittest discover -p 'test_*.py'
USER 65532:65532
ENTRYPOINT ["python", "/app/remind.py"]
