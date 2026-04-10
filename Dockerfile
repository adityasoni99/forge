FROM node:22-bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends git make netcat-openbsd && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /opt/forge/harness

COPY harness/package.json harness/package-lock.json /opt/forge/harness/
WORKDIR /opt/forge/harness
RUN npm ci --production

COPY harness/dist/ /opt/forge/harness/dist/
COPY scripts/sandbox-entry.sh /usr/local/bin/sandbox-entry.sh
RUN chmod +x /usr/local/bin/sandbox-entry.sh

ARG TARGETARCH=amd64
COPY forge-linux-${TARGETARCH} /usr/local/bin/forge
RUN chmod +x /usr/local/bin/forge

WORKDIR /workspace
ENTRYPOINT ["sandbox-entry.sh"]
