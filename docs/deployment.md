# CloudEmu 部署文档

## 前置条件

| 要求 | 版本/说明 |
|------|----------|
| 操作系统 | Ubuntu 24.04（libretro .so 需要 GLIBC ≥ 2.38） |
| Docker | ≥ 20.x，`docker run` 可用 |
| docker-compose | 1.29.2+ |
| 公网 IP | 需绑定到服务器（LiveKit TURN 会通告此 IP 给客户端） |
| 磁盘空间 | ≥ 10GB（镜像 + 数据卷） |

### 防火墙端口清单

| 端口 | 协议 | 服务 | 说明 |
|------|------|------|------|
| 80 | TCP | nginx（前端） | 对外唯一入口 |
| 3478 | TCP/UDP | coturn | TURN/STUN（WiFi 客户端） |
| 443 | TCP | coturn | TURN/TLS（4G 客户端 fallback） |
| 7880 | TCP | LiveKit | 信令服务（需对 Worker 和前端可达） |
| 7881 | TCP | LiveKit | WebRTC TCP fallback |
| 40000-49000 | UDP | coturn | TURN 媒体中继端口（仅本机内部使用，**无需**在安全组开放） |
| 50000-60000 | UDP | LiveKit | RTC 直连端口（仅本机内部使用，**无需**在安全组开放） |

> `8080`（Control Plane）、`5432`（PostgreSQL）、`6379`（Redis）、`9000`（MinIO）等内部服务端口**不对外开放**，仅在 Docker 网络内通过 nginx 反向代理访问。

---

## 目录与文件速览

```
deploy/
├── .env                  # 【必须修改】所有环境变量集中管理
├── docker-compose.yaml   # 7 个服务定义（PG/Redis/MinIO/LiveKit/coturn/CP/Worker/Web）
├── livekit.yaml          # LiveKit 服务器配置（信令 + RTC + TURN 中继）
├── turnserver.conf       # coturn TURN 配置
├── turn-cert.pem         # 【需生成】TURN/TLS 自签名证书
├── turn-key.pem          # 【需生成】TURN/TLS 私钥
```

---

## 1. 生成 TURN/TLS 自签名证书

LiveKit 的外部 coturn 需要 TLS 证书才能在 4G 移动网络下工作（运营商普遍阻断 UDP，TURN/TLS 走 TCP 443 不被拦截）。

```bash
# 将 IP 替换为你的服务器公网 IP
openssl req -x509 -newkey rsa:2048 \
  -keyout deploy/turn-key.pem \
  -out deploy/turn-cert.pem \
  -days 3650 -nodes \
  -subj "/CN=<你的公网IP>"

# 修复权限：coturn 容器内若 key 文件权限为 0600 会导致 TLS 无法启动
chmod 644 deploy/turn-key.pem
```

> 浏览器对 TURN 服务器的证书验证较宽松，自签名证书可用于生产环境。

---

## 2. 配置 `.env` 文件

所有服务的关键配置集中在此文件。**部署前必须替换标记为「必须修改」的项。**

### 对外暴露端口

```bash
WEB_PORT=80           # 前端 nginx
CP_PORT=8080          # Control Plane API（调试用，可注释掉不开放）
PG_PORT=5432          # PostgreSQL（调试用，可注释掉不开放）
REDIS_PORT=6379       # Redis（调试用，可注释掉不开放）
MINIO_PORT=9000       # MinIO API（调试用，可注释掉不开放）
MINIO_CONSOLE_PORT=9001  # MinIO Web 控制台
```

### PostgreSQL

```bash
PG_USER=postgres
PG_PASSWORD=<必须修改>
PG_DB=cloudemu
```

### Redis

```bash
REDIS_PASS=<必须修改>        # 与 PG_PASSWORD 可不同
WORKER_REDIS_DB=1            # Worker 注册专用 DB，勿改
```

### JWT

```bash
JWT_SECRET=<必须修改>        # 用 openssl rand -hex 32 生成
```

### MinIO

```bash
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=<必须修改>
MINIO_BUCKET=cloudemu
```

### 前端

```bash
FRONTEND_BASE_URL=http://<你的公网IP或域名>
```

### SMTP（可选，不配置则禁用邮件功能）

```bash
SMTP_HOST=smtp.qq.com
SMTP_PORT=465
SMTP_USER=<你的邮箱>
SMTP_PASS=<邮箱授权码>
SMTP_FROM=<你的邮箱>
SMTP_USE_TLS=true
```

### Worker

```bash
WORKER_ADDR=:9090                        # Worker 内部 gRPC 监听
LIVEKIT_HOST=http://<你的公网IP>:7880    # 必须与 livekit.yaml 中 keys 一致
LIVEKIT_API_KEY=<必须与 livekit.yaml 一致>
LIVEKIT_API_SECRET=<必须与 livekit.yaml 一致>
```

### TURN

```bash
TURN_EXTERNAL_IP=<你的公网IP>        # coturn 通告给客户端的地址
TURN_SECRET=<必须修改>               # 【关键】必须与 livekit.yaml 中 turn_servers[].secret 一致
```

### 跨文件一致性检查清单

| 值 | 出现在 | 说明 |
|------|------|------|
| `LIVEKIT_API_KEY` / `LIVEKIT_API_SECRET` | `.env` + `livekit.yaml:keys` | 两处必须相同 |
| `TURN_SECRET` | `.env` + `livekit.yaml:turn_servers[].secret` | 两处必须相同 |
| 公网 IP | `.env` + `livekit.yaml:turn_servers[].host` + 证书 CN | 三处必须相同 |

---

## 3. 配置文件说明

### 3.1 `livekit.yaml` — LiveKit 服务

```yaml
port: 7880                              # 信令端口
rtc:
  tcp_port: 7881                        # WebRTC TCP fallback
  port_range_start: 50000               # RTC UDP 直连端口范围
  port_range_end: 60000
  use_external_ip: true                 # 自动通过 STUN 发现公网 IP
  turn_servers:                         # 【关键】外部 coturn TURN 中继
    - host: <公网IP>
      port: 3478
      protocol: udp                     # WiFi 客户端使用 UDP TURN
      secret: <必须与 .env TURN_SECRET 一致>
      ttl: 14400                        # 凭证有效期（秒）
    - host: <公网IP>
      port: 443
      protocol: tls                     # 4G 客户端使用 TLS TURN（自签名证书）
      secret: <同上>
      ttl: 14400
keys:
  <LIVEKIT_API_KEY>: <LIVEKIT_API_SECRET>  # 必须与 .env 一致
turn:
  enabled: false                        # 【重要】关闭 LiveKit 内置 TURN，使用外部 coturn
```

**设计说明：**

- `turn.enabled: false` — LiveKit 自带内置 TURN 但不使用。外部 coturn 通过 `rtc.turn_servers` 集成。
- `rtc.turn_servers` 配置了 UDP（3478）和 TLS（443）双协议。浏览器 ICE 协商时自动选择可用协议：WiFi 优先 UDP，4G fallback 到 TLS。
- `use_external_ip: true` — 云服务器上内网 IP 与公网 IP 不同，LiveKit 自动通过 STUN 发现并通告公网 IP。
- `port_range_start/end` 为 LiveKit 自身 RTC 端口范围（50000-60000），与 coturn relay 端口范围（40000-49000）不重叠，避免冲突。

### 3.2 `turnserver.conf` — coturn TURN 中继

```conf
listening-port=3478                     # 标准 STUN/TURN UDP 端口
tls-listening-port=443                 # TURN/TLS TCP 端口
cert=/etc/turn-cert.pem                 # 自签名证书（容器内挂载路径）
pkey=/etc/turn-key.pem                  # 私钥
use-auth-secret                         # 使用共享密钥认证（非静态用户名密码）
realm=cloudemu
min-port=40000                          # TURN relay 端口范围
max-port=49000                          # 与 LiveKit RTC（50000-60000）不重叠
verbose
fingerprint
lt-cred-mech                            # 支持长期凭证机制（兼容 LiveKit 限时 token）
no-multicast-peers
```

**设计说明：**

- `use-auth-secret` + `lt-cred-mech` — LiveKit 用预共享密钥生成限时 TURN 凭证（格式：`{timestamp}:{username}`）。客户端不持有原始密钥，凭证过期自动失效。
- `static-auth-secret` 不在配置文件中，通过 docker-compose 的 `command:` 参数 `--static-auth-secret ${TURN_SECRET}` 从 `.env` 注入，避免明文落盘。
- `min-port/max-port` 设置为 40000-49000，与 LiveKit RTC 端口范围（50000-60000）**不重叠**，避免 host 网络下端口冲突。
- `network_mode: host` — coturn 必须看到公网 IP 才能正确通告 ICE candidate。host 网络下无需 `ports:` 映射。
- `cert/pkey` 由 docker-compose volumes 挂载，文件权限需 `644`（coturn 容器内非 root 用户需要可读）。

### 3.3 `docker-compose.yaml` — 服务编排

#### coturn 特殊处理

coturn 配置在 compose 文件中，但 docker-compose 1.29.2 对该镜像存在 `ContainerConfig` 解析 bug，导致 `up` 重建容器时报 KeyError。当前使用 **`docker run` 手动管理**：

```bash
# 首次启动或重建 coturn：
docker rm -f cloudemu-coturn 2>/dev/null
docker run -d \
  --name cloudemu-coturn \
  --network host \
  --restart unless-stopped \
  --cap-add=NET_BIND_SERVICE \
  -v $(pwd)/deploy/turnserver.conf:/etc/turnserver.conf:ro \
  -v $(pwd)/deploy/turn-cert.pem:/etc/turn-cert.pem:ro \
  -v $(pwd)/deploy/turn-key.pem:/etc/turn-key.pem:ro \
  coturn/coturn:4.14.0 \
  -c /etc/turnserver.conf \
  --static-auth-secret <TURN_SECRET> \
  --external-ip <公网IP>
```

> `--cap-add=NET_BIND_SERVICE` 是必须的：coturn 容器以非 root 用户运行，需要此 Linux capability 才能绑定特权端口 443（< 1024）。

#### LiveKit 健康检查

LiveKit 镜像不含 `curl`，但包含 `wget`。健康检查使用 `wget --spider -q`：

```yaml
healthcheck:
  test: ["CMD", "wget", "--spider", "-q", "http://localhost:7880/"]
```

#### Control Plane 健康检查

Control Plane 暴露了独立的 `/health` 端点（`internal/control-plane/router/router.go`），curl 检查：

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
```

#### Worker 网络

Worker 的 gRPC 端口（9090）**不映射到宿主机**。Control Plane 通过 Docker 内部 DNS（`worker:9090`）直连，不经过公网。

---

## 4. 分步部署

假设项目根目录在 `/home/ubuntu/cloudemu`，compose 文件在 `deploy/` 子目录下。

### 4.1 准备环境变量

```bash
cd /home/ubuntu/cloudemu

# 生成密钥（用于 JWT_SECRET 和 TURN_SECRET）
openssl rand -hex 32

# 编辑 .env，替换所有 <必须修改> 的值
vim deploy/.env
```

### 4.2 生成 TURN 证书

```bash
# 替换 <你的公网IP>
openssl req -x509 -newkey rsa:2048 \
  -keyout deploy/turn-key.pem \
  -out deploy/turn-cert.pem \
  -days 3650 -nodes \
  -subj "/CN=<你的公网IP>"

chmod 644 deploy/turn-key.pem
```

### 4.3 同步跨文件密钥

确保以下值在 `.env` 和 `livekit.yaml` 中完全一致：

```bash
# 1. LIVEKIT_API_KEY / LIVEKIT_API_SECRET
grep LIVEKIT_API deploy/.env
grep -A2 "^keys:" deploy/livekit.yaml | tail -2

# 2. TURN_SECRET
grep TURN_SECRET deploy/.env
grep "secret:" deploy/livekit.yaml

# 3. 公网 IP
grep "TURN_EXTERNAL_IP\|FRONTEND_BASE_URL\|LIVEKIT_HOST" deploy/.env
grep "host:" deploy/livekit.yaml
cat deploy/turn-cert.pem | openssl x509 -noout -subject
```

### 4.4 启动基础服务

```bash
# 启动数据库 + 缓存 + 存储
docker-compose -f deploy/docker-compose.yaml up -d postgres redis minio

# 等待健康检查通过
docker-compose -f deploy/docker-compose.yaml ps
```

### 4.5 启动 Control Plane

```bash
# 构建 + 启动
docker-compose -f deploy/docker-compose.yaml up -d --build controlplane

# 确认日志无错误
docker logs cloudemu-cp
```

### 4.6 启动 LiveKit

```bash
docker-compose -f deploy/docker-compose.yaml up -d livekit

# 确认正常
docker logs cloudemu-livekit | tail -5
curl http://localhost:7880/
# 应返回 "OK"
```

### 4.7 启动 coturn（compose bug 手工处理）

```bash
# 先清理可能存在的旧容器
docker rm -f cloudemu-coturn 2>/dev/null

# 从 .env 读取敏感值
source deploy/.env

# 启动
docker run -d \
  --name cloudemu-coturn \
  --network host \
  --restart unless-stopped \
  --cap-add=NET_BIND_SERVICE \
  -v $(pwd)/deploy/turnserver.conf:/etc/turnserver.conf:ro \
  -v $(pwd)/deploy/turn-cert.pem:/etc/turn-cert.pem:ro \
  -v $(pwd)/deploy/turn-key.pem:/etc/turn-key.pem:ro \
  coturn/coturn:4.14.0 \
  -c /etc/turnserver.conf \
  --static-auth-secret "${TURN_SECRET}" \
  --external-ip "${TURN_EXTERNAL_IP}"

# 验证
ss -tlnp | grep -E "3478|443"
# 应看到 TCP 3478 和 TCP 443 在两个端口上监听
```

### 4.8 启动 Worker + 前端

```bash
docker-compose -f deploy/docker-compose.yaml up -d --build worker web

# 查看整体状态
docker-compose -f deploy/docker-compose.yaml ps
```

---

## 5. 验证清单

### 5.1 各服务健康状态

```bash
docker-compose -f deploy/docker-compose.yaml ps
# 所有服务 STATE 应为 Up (healthy)
```

### 5.2 Control Plane

```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```

### 5.3 LiveKit

```bash
curl http://localhost:7880/
# → OK
```

### 5.4 coturn

```bash
# UDP 端口监听
ss -ulnp | grep 3478

# TLS 端口监听
ss -tlnp | grep 443

# 日志
docker logs cloudemu-coturn 2>&1 | tail -10
# 应看到 "TLS 1.2 supported" / "TLS 1.3 supported"
```

### 5.5 前端

```bash
curl http://localhost/ -o /dev/null -s -w "%{http_code}"
# → 200
```

### 5.6 从浏览器测试

1. 访问 `http://<公网IP>`，确认登录注册页面正常加载
2. 在 WiFi 下创建房间并进入游戏
3. 切换到 4G 移动网络，确认仍能连接
4. 查看 coturn 日志确认 TURN/TLS 分配生效：
   ```bash
   docker logs cloudemu-coturn 2>&1 | grep "TCP/TLS" | tail -5
   ```

---

## 6. 常见问题

### PG 密码错误

**现象**：Control Plane 日志 `failed to connect database`，但 `.env` 密码和 `POSTGRES_PASSWORD` 相同。

**原因**：PostgreSQL 数据卷已有旧密码初始化的数据，`POSTGRES_PASSWORD` 环境变量只在**首次初始化**时生效。

**解决**：

```bash
docker-compose -f deploy/docker-compose.yaml down
docker volume rm deploy_pgdata
docker-compose -f deploy/docker-compose.yaml up -d postgres
```

> 这会清空所有数据，仅适用于开发环境。

### 4G 移动网络无法连接

**现象**：WiFi 正常，4G 下 WebRTC 连接失败（ICE 协商超时）。

**诊断**：

```bash
# 查看 coturn 日志，关注 "remote" IP 和 "allocation timeout"
docker logs cloudemu-coturn 2>&1 | grep -E "allocation|timeout|remote 111|remote 112"
# 若远程 IP 以 111 或 112 开头（移动 4G 网段），且出现 allocation timeout → UDP 被运营商阻断
```

**解决**：确保 TURN/TLS 已配置（见 §1 和 §3.2），且云服务商安全组已开放 **TCP 443**。

### coturn TLS 端口未监听

**现象**：`ss -tlnp | grep 443` 无输出。

**检查**：

```bash
# 查看 coturn 日志中的 TLS 相关信息
docker logs cloudemu-coturn 2>&1 | grep -i "tls\|key file\|private key\|cert"
```

常见原因：
1. **key 文件权限**：需 `chmod 644 deploy/turn-key.pem`
2. **证书文件路径**：确认 `docker run -v` 挂载的宿主机路径正确
3. **证书与 CN 不匹配**：subject CN 需与 `--external-ip` 一致

### docker-compose 重建 coturn 报 ContainerConfig KeyError

**现象**：`docker-compose up -d coturn` 报 `KeyError: 'ContainerConfig'`。

**原因**：docker-compose 1.29.2 对 `coturn/coturn:4.14.0` 镜像解析 bug。

**解决**：使用 `docker run` 手动管理（见 §4.7）。

### LiveKit 健康检查始终不通过

**现象**：`curl http://localhost:7880/` 返回 OK，但 compose 健康检查失败。

**原因**：LiveKit 镜像不含 `curl`。

**解决**：健康检查用 `wget`（已配置）：
```yaml
test: ["CMD", "wget", "--spider", "-q", "http://localhost:7880/"]
```

### 前端 404（JS/CSS 资源）

**现象**：页面加载但 JS/CSS 资源 404。

**原因**：`nginx.conf` 中 `root` 指令在 `location /` 内，`/assets/` 块未继承导致路径错误。

**解决**：已在 `web/nginx.conf` 中将 `root` 提升到 `server` 级别，并给 `/assets/` 添加 `try_files $uri =404`。

### Worker 在 Docker 中的地址通告

**现象**：Control Plane 无法连接 Worker gRPC。

**原因**：Worker 向 Redis 注册的地址是 `WORKER_ADDR` 环境变量的值（当前为 `worker:9090`），Docker DNS 在 compose 网络内可解析。

**确认**：

```bash
docker exec cloudemu-cp wget -q -O- http://worker:9090 || echo "unreachable"
# 若 unreachable，检查 WORKER_ADDR 和容器网络
```

---

## 7. 日常运维

### 查看全部日志

```bash
# 所有服务
docker-compose -f deploy/docker-compose.yaml logs --tail=50

# 单服务
docker logs -f cloudemu-coturn
```

### 更新配置后重启

```bash
# 修改 livekit.yaml 后
docker restart cloudemu-livekit

# 修改 .env 后（Control Plane 重读环境变量需重建）
docker-compose -f deploy/docker-compose.yaml up -d --build controlplane

# 修改 turnserver.conf 后（coturn 需重建）
docker rm -f cloudemu-coturn
# 重新执行 §4.7 的 docker run 命令
```

### 清理与重建

```bash
# 完全清理（含数据卷）
docker-compose -f deploy/docker-compose.yaml down -v
docker rm -f cloudemu-coturn 2>/dev/null

# 重建所有
# 按 §4.4 → §4.8 顺序执行
```
