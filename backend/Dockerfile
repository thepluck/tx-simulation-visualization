FROM golang:1.22-bookworm AS builder

WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/txsim-server ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update \
  && apt-get install -y --no-install-recommends bash ca-certificates curl git gzip tar \
  && rm -rf /var/lib/apt/lists/*

RUN curl -L https://foundry.paradigm.xyz | bash \
  && /root/.foundry/bin/foundryup

ENV PATH="/root/.foundry/bin:${PATH}"
ENV TXSIM_CONFIG="/app/backend/config.docker.yaml"

WORKDIR /app
COPY --from=builder /out/txsim-server /usr/local/bin/txsim-server
COPY backend/config.docker.yaml ./backend/config.docker.yaml
COPY contracts ./contracts
COPY foundry.toml foundry.lock remappings.txt ./
COPY lib ./lib
RUN mkdir -p /data/runs

EXPOSE 8080
CMD ["txsim-server"]
