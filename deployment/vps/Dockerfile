FROM ubuntu:24.04
ENV DEBIAN_FRONTEND=noninteractive TZ=Asia/Ho_Chi_Minh
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libc++1 libc++abi1 libopus0 libopusfile0 ffmpeg curl tzdata && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY bundle/ /app/
RUN chmod 755 /app/xiaozhi_server && useradd --uid 10001 --no-create-home xiaozhi && mkdir -p /app/data /app/logs /app/storage && chown -R 10001:10001 /app
ENV LD_LIBRARY_PATH=/app/ten-vad/lib/Linux/x64 GOMAXPROCS=4
USER 10001:10001
CMD ["./xiaozhi_server", "-c", "/app/config/main_config.yaml", "-manager-enable=true", "-manager-config", "/app/config/manager.json", "-asr-enable=false"]
