# syntax=docker/dockerfile:1.7

FROM node:24.18.0-alpine AS frontend
WORKDIR /src/frontend
ARG NPM_REGISTRY=https://registry.npmjs.org
COPY frontend/package.json frontend/package-lock.json ./
RUN --mount=type=cache,id=fribench-monitor-npm,target=/root/.npm,sharing=locked \
    npm ci --registry="${NPM_REGISTRY}"
COPY frontend/ ./
RUN npm run build

FROM golang:1.25.6-alpine AS backend
WORKDIR /src
COPY go.mod ./
COPY cmd/ cmd/
COPY internal/ internal/
COPY --from=frontend /src/internal/webui/dist/ internal/webui/dist/
RUN go test ./... && go vet ./...
RUN --mount=type=cache,id=fribench-monitor-go-build,target=/root/.cache/go-build,sharing=locked \
    CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/fribench-monitor \
      ./cmd/fribench-monitor

FROM scratch AS runtime
COPY --from=backend /out/fribench-monitor /fribench-monitor
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/fribench-monitor"]
