FROM python:3.9-alpine

WORKDIR /home

ADD ./requirements.txt ./requirements.txt
RUN python3 -m venv venv && source venv/bin/activate
RUN pip install -r requirements.txt
ADD . .

ENV FLASK_RUN_PORT=8000
ENV FLASK_ENV=development
ENV PYTHONUNBUFFERED=1

ENTRYPOINT ["python3", "./server.py"]
