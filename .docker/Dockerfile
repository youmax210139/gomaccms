# 构建上下文为仓库根目录:  docker build -f .docker/Dockerfile .
# (根目录 docker-compose.yml 中已配置)

# ---------- 1. 构建管理后台 (Inertia + Vue, Vite) ----------
# vite.config.ts 将产物输出到仓库根目录的 public/build
FROM node:22-alpine AS admin
WORKDIR /src/admin
COPY admin/package.json ./
RUN --mount=type=cache,target=/root/.npm npm install --no-audit --no-fund
COPY admin/ ./
RUN npm run build

# ---------- 2. 构建 Go 程序 ----------
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd ./cmd
COPY internal ./internal
# TARGETARCH 由 buildx 自动注入, amd64 / arm64 服务器均可直接构建
ARG TARGETOS=linux
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/gomaccms ./cmd/server

# ---------- 3. 运行镜像 ----------
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai \
    GIN_MODE=release
WORKDIR /app
# 程序运行时按相对路径读取 themes/、admin/root.html 与 public/build, 工作目录必须是 /app
COPY --from=builder /out/gomaccms ./gomaccms
COPY themes ./themes
COPY admin/root.html ./admin/root.html
COPY --from=admin /src/public/build ./public/build
# 上传/同步的图片 (config.FilmPictureUploadDir = ./storage/upload/gallery)
VOLUME ["/app/storage"]
EXPOSE 3601
ENTRYPOINT ["./gomaccms"]
