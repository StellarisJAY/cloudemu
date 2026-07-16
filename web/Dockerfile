# CloudEmu 前端多阶段构建
# Stage 1: pnpm + Vite 构建 Vue SPA
# Stage 2: nginx 托管静态文件 + 反向代理 /api 到 Control Plane

# ─── 第一阶段：构建 ───
FROM node:22-alpine AS builder

WORKDIR /build

# 启用 pnpm（Node 22+ 内置 Corepack）
RUN corepack enable && corepack prepare pnpm@latest --activate

# 安装依赖（利用 Docker 层缓存）
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# 复制源码并构建
COPY web/ ./

# 注入生产环境 API 路径（Vite 编译时静态替换 import.meta.env.VITE_API_BASE）
ENV VITE_API_BASE=/api

# 跳过 type-check，仅执行 Vite 构建（生产环境类型检查应在 CI 完成）
RUN pnpm build-only

# ─── 第二阶段：运行时 ───
FROM nginx:alpine

# 复制构建产物到 nginx 静态目录
COPY --from=builder /build/dist /usr/share/nginx/html

# 复制 nginx 配置
COPY web/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
