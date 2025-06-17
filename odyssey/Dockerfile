FROM ubuntu:22.04 AS builder

# Установка зависимостей для сборки
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    build-essential \
    cmake \
    git \
    libpq-dev \
    libssl-dev \
    postgresql-server-dev-14 \
    && rm -rf /var/lib/apt/lists/*

# Клонирование и сборка Odyssey
RUN git clone --branch 1.4rc https://github.com/yandex/odyssey.git /odyssey \
    && cd /odyssey \
    && make local_build

# Финальный образ
FROM ubuntu:22.04

# Runtime зависимости
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    libssl3 \
    libpq5 \
    postgresql-client-14 \
    && rm -rf /var/lib/apt/lists/*

# Копируем бинарник Odyssey
COPY --from=builder /odyssey/build/sources/odyssey /usr/bin/odyssey

CMD ["odyssey", "/etc/odyssey/odyssey.conf"]