FROM python:3-alpine

EXPOSE 8080

WORKDIR /root/YtbDownBot

COPY src ./
COPY requirements.txt ./
COPY start.sh ./

ADD youtubedl-autoupdate /etc/periodic/hourly/youtubedl

ENV LIBRARY_PATH=/lib:/usr/lib

RUN apt update && \
    apt add --no-cache curl ffmpeg libwebp && \
    apt add --no-cache --virtual .build-deps git gcc musl-dev libffi-dev build-base python3-dev jpeg-dev zlib-dev libwebp-dev && \
    pip3 install --no-cache-dir -r requirements.txt  && \
    apk del .build-deps && \
    chmod +x ./start.sh && \
    chmod +x /etc/periodic/hourly/youtubedl && \
    touch /var/log/cron.log && \
    rm -rf /var/cache/apk/*

CMD ["./start.sh"]
