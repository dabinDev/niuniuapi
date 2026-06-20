# Tomato 本地 Docker 构建发布流程

本文档用于发布 `tomato` / `番茄` 这一套独立服务。目标是：**在本机完成 Docker 镜像构建，线上服务器只加载镜像并重启新容器，避免再次卡住原有 Sub2 服务。**

## 服务器与路径

本地项目目录：

```powershell
E:\ForkProject\niuniuapi
```

当前线上服务器：

```text
IP: 47.86.203.13
SSH 用户: root
推荐登录方式: ssh -F NUL -i C:/Users/dabin/.ssh/id_rsa root@47.86.203.13
域名: tomato.beinai.cc
```

注意：不要把私钥文件内容、服务器密码写进仓库，也不要提交 `.env`、证书私钥、数据库密码。

线上新服务目录：

```text
/opt/niuniuapi
/opt/niuniuapi/deploy
/opt/niuniuapi/deploy/.env
/opt/niuniuapi/deploy/docker-compose.tomato.yml
```

线上临时镜像包上传位置：

```text
/tmp/tomato-release.tar
```

Nginx 当前站点配置：

```text
/etc/nginx/conf.d/tomato.beinai.cc.conf
```

## 绝对不要影响旧 Sub2

旧 Sub2 正在同一台服务器运行，发布 `tomato` 时不要操作这些资源：

```text
旧目录: /opt/sub2api
旧应用容器: sub2api
旧数据库容器: sub2api-postgres
旧 Redis 容器: sub2api-redis
旧端口: 18080
旧域名: token.cylonai.cn, sub.cyroute.cn
```

新 `tomato` 必须使用这些独立资源：

```text
新目录: /opt/niuniuapi
新应用容器: tomato
新数据库容器: tomato-postgres
新 Redis 容器: tomato-redis
新端口: 18089
新域名: tomato.beinai.cc
```

发布时禁止执行：

```bash
docker restart sub2api
docker compose -f /opt/sub2api/docker-compose.yml up -d
docker compose -f /opt/sub2api/docker-compose.yml down
rm -rf /opt/sub2api
```

## 本机构建镜像

在 Windows PowerShell 执行：

```powershell
cd E:\ForkProject\niuniuapi

git status --short --branch
git pull --ff-only

docker build -t tomato:latest .
```

如果本机 Docker 访问 Docker Hub 很慢或失败，可以使用镜像源构建。这个命令只在本机运行，不会碰线上服务器：

```powershell
docker build --progress=plain `
  --build-arg NODE_IMAGE=docker.1ms.run/library/node:24-alpine `
  --build-arg GOLANG_IMAGE=docker.1ms.run/library/golang:1.26.4-alpine `
  --build-arg ALPINE_IMAGE=docker.1ms.run/library/alpine:3.21 `
  --build-arg POSTGRES_IMAGE=docker.1ms.run/library/postgres:18-alpine `
  -t tomato:latest .
```

如果需要带版本标签，推荐同时打一个时间戳标签：

```powershell
$tag = "tomato-" + (Get-Date -Format "yyyyMMddHHmm")
docker build -t "tomato:$tag" -t tomato:latest .
```

构建完成后本地验证镜像存在：

```powershell
docker images tomato --format 'table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.CreatedSince}}\t{{.Size}}'
```

如果先构建了测试标签，例如 `tomato:local-test`，发布前再改成正式标签：

```powershell
docker tag tomato:local-test tomato:latest
```

排障建议：

```text
1. 构建卡住时先加 --progress=plain，看具体卡在哪一步。
2. 只检查本机 docker-buildx / Docker Desktop，不要登录服务器处理构建问题。
3. 不要从生产服务器 docker save 或 docker pull 镜像回来，本地构建失败就继续修本地环境。
```

## 导出镜像包

```powershell
docker save tomato:latest -o C:\Users\dabin\AppData\Local\Temp\tomato-release.tar
```

确认文件生成：

```powershell
Get-Item C:\Users\dabin\AppData\Local\Temp\tomato-release.tar
```

## 上传镜像到服务器

```powershell
scp -F NUL -i C:/Users/dabin/.ssh/id_rsa `
  C:/Users/dabin/AppData/Local/Temp/tomato-release.tar `
  root@47.86.203.13:/tmp/tomato-release.tar
```

如果只改了 compose 文件，也可以单独上传 compose 到新服务目录，注意路径是 `/opt/niuniuapi`，不是 `/opt/sub2api`：

```powershell
scp -F NUL -i C:/Users/dabin/.ssh/id_rsa `
  E:/ForkProject/niuniuapi/deploy/docker-compose.tomato.yml `
  root@47.86.203.13:/opt/niuniuapi/deploy/docker-compose.tomato.yml
```

## 服务器加载镜像并重启新服务

登录服务器：

```powershell
ssh -F NUL -i C:/Users/dabin/.ssh/id_rsa root@47.86.203.13
```

在服务器执行：

```bash
docker load -i /tmp/tomato-release.tar
docker images tomato

cd /opt/niuniuapi/deploy
docker compose -f docker-compose.tomato.yml --env-file .env up -d --no-build tomato
```

关键点：

```text
必须加 --no-build
```

这样服务器只会使用已经上传的镜像，不会在生产服务器上重新跑 `npm install`、`vite build`、`go build`，避免把旧 Sub2 卡住。

## 发布后验证

检查新旧容器状态：

```bash
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}' | egrep 'NAMES|tomato|sub2api|nginx-token'
```

期望看到：

```text
tomato               Up ... (healthy)   0.0.0.0:18089->8080/tcp
tomato-postgres      Up ... (healthy)   5432/tcp
tomato-redis         Up ...             6379/tcp
sub2api              Up ... (healthy)   0.0.0.0:18080->8080/tcp
sub2api-postgres     Up ... (healthy)   5432/tcp
sub2api-redis        Up ...             6379/tcp
```

检查新服务本机端口：

```bash
curl -sS -m 8 -I http://127.0.0.1:18089/home | head
curl -sS -m 8 http://127.0.0.1:18089/health
```

检查旧 Sub2 没受影响：

```bash
curl -sS -m 8 -I http://127.0.0.1:18080/home | head
```

检查 Nginx 配置：

```bash
docker exec nginx-token nginx -t
```

模拟域名访问新服务：

```bash
curl -sS -m 8 -I -H 'Host: tomato.beinai.cc' http://127.0.0.1/home | head
```

本机 Windows 也可以模拟 DNS 访问：

```powershell
curl.exe -I --max-time 20 http://tomato.beinai.cc/
curl.exe -sS --max-time 20 https://tomato.beinai.cc/health
```

## DNS 与 SSL

当前线上域名为 `tomato.beinai.cc`，A 记录指向 `47.86.203.13`。

当前 `18089` 不需要对公网开放。推荐只让 Nginx 通过 80/443 访问新服务：

```text
公网: 80/443
内网/宿主机: 18089
```

HTTPS 使用 `tomato.beinai.cc` 自己的 SSL 证书。不能直接复用其它域名证书，否则浏览器会提示证书域名不匹配。

证书放置建议：

```text
/etc/nginx/ssl/tomato.beinai.cc/tomato.beinai.cc_bundle.crt
/etc/nginx/ssl/tomato.beinai.cc/tomato.beinai.cc.key
```

证书配置完成后，再把 `/etc/nginx/conf.d/tomato.beinai.cc.conf` 增加 443 server，并执行：

```bash
nginx -t
systemctl reload nginx
```

## 回滚

如果新镜像启动失败，但旧 Sub2 正常，不要动旧 Sub2。只处理新服务：

```bash
cd /opt/niuniuapi/deploy
docker compose -f docker-compose.tomato.yml --env-file .env logs --tail=200 tomato
docker compose -f docker-compose.tomato.yml --env-file .env restart tomato
```

如果需要回滚到上一版镜像，先确认本机或服务器保留了旧镜像标签，例如：

```bash
docker images tomato
docker tag tomato:上一版本 tomato:latest
cd /opt/niuniuapi/deploy
docker compose -f docker-compose.tomato.yml --env-file .env up -d --no-build tomato
```

如果只是新站完全不要对外，停新服务即可：

```bash
cd /opt/niuniuapi/deploy
docker compose -f docker-compose.tomato.yml --env-file .env stop tomato
```

不要执行 `down -v`，否则会删除新服务的数据卷。

## 本次踩坑记录

1. 不要在生产服务器上完整构建镜像。前端 `npm install`、`vue-tsc`、`vite build` 和 Go 编译会抢 CPU/IO，旧 Sub2 会变慢。
2. 首次远程构建日志显示前端 Vite 构建约 5 分 46 秒，Go 编译约 168 秒，这些都应放到本机 Docker 环境完成。
3. Docker 构建时 `.dockerignore` 会过滤 build context。本项目的前端会 import `docs/legal/*.md?raw`，所以 `.dockerignore` 必须保留：

```text
docs/
!docs/
!docs/legal/
!docs/legal/**
```

4. 当前 Dockerfile 使用 `npm install --no-package-lock`，是因为服务器 Docker 环境中 `pnpm` 曾出现 `EPERM: operation not permitted, write`。后续如果要恢复 pnpm，需要先在本机和服务器 Docker build 中完整验证。
5. 服务器 compose 启动时使用 `--no-build`，这是防止再次触发线上构建的关键。
6. 2026-06-13 本机 Docker 已验证通过一次镜像源构建，命令使用 `docker.1ms.run` 和 `--progress=plain`，生成镜像：

```text
REPOSITORY   TAG                 IMAGE ID       SIZE
tomato       local-test   7851fa1c445d   144MB
```

这次没有发布、没有登录服务器、没有从服务器拉取或导出镜像。

## tomato.beinai.cc Nginx/SSL 发布流程（2026-06-19）

本节记录 `tomato.beinai.cc` 子域名在 `47.86.203.13` 服务器上的 Nginx/SSL 配置流程。目标是让 `http://tomato.beinai.cc` 自动跳转到 HTTPS，并且 HTTPS 使用 `tomato.beinai.cc` 自己的证书，不影响同机已有的 `qbook.top`。

### 服务器与现状

```text
服务器 IP: 47.86.203.13
SSH 用户: root
登录方式: 密码登录（密码不要写入仓库或记忆）
推荐登录方式: `ssh -F NUL -i C:/Users/dabin/.ssh/id_rsa root@47.86.203.13`
Nginx: nginx/1.24.0 (Ubuntu)
应用上游: 127.0.0.1:18089 -> tomato 容器 8080
域名: tomato.beinai.cc -> 47.86.203.13
```

同机已有 `qbook.top`，配置在：

```text
/etc/nginx/sites-enabled/qbook.top
/etc/letsencrypt/live/qbook.top/fullchain.pem
/etc/letsencrypt/live/qbook.top/privkey.pem
```

发布 `tomato.beinai.cc` 时不要修改 qbook 的配置文件、证书目录或 server block。只新增/覆盖 tomato 自己的文件：

```text
/etc/nginx/conf.d/tomato.beinai.cc.conf
/etc/nginx/ssl/tomato.beinai.cc/tomato.beinai.cc_bundle.crt
/etc/nginx/ssl/tomato.beinai.cc/tomato.beinai.cc.key
```

### 本地准备文件

SSL 压缩包来源：

```text
G:/my-linux/tomato.beinai.cc_nginx.zip
```

本地解压位置：

```text
E:/ForkProject/niuniuapi/deploy/ssl/tomato.beinai.cc/
```

证书文件：

```text
tomato.beinai.cc_bundle.crt
tomato.beinai.cc_bundle.pem
tomato.beinai.cc.key
tomato.beinai.cc.csr
```

`deploy/ssl/` 已加入 `deploy/.gitignore`，不要提交证书私钥。

Nginx 配置与脚本：

```text
deploy/nginx/tomato.beinai.cc.conf
deploy/nginx/install-tomato-beinai-nginx.sh
deploy/nginx/deploy-tomato-beinai-from-windows.ps1
```

### 从 Windows 上传并安装

如果可以用 OpenSSH 密钥登录，可用 PowerShell 脚本：

```powershell
.\deploy\nginx\deploy-tomato-beinai-from-windows.ps1 `
  -Server 47.86.203.13 `
  -User root `
  -IdentityFile "你的 pem 路径"
```

如果只有密码登录，使用 PuTTY 工具时要固定 host key，避免误连：

```powershell
$remote = "root@47.86.203.13"
$hostkey = "SHA256:wzD4VaS8pEgIt4Fd/5uQrrKajBLPJgHGLvyJYj4U4Bo"
$pw = "从安全位置读取，不要写入文档"

& "C:\Program Files\PuTTY\plink.exe" -ssh $remote -hostkey $hostkey -pw $pw -batch `
  "rm -rf /tmp/tomato-beinai-nginx && mkdir -p /tmp/tomato-beinai-nginx/deploy/nginx /tmp/tomato-beinai-nginx/deploy/ssl/tomato.beinai.cc"

& "C:\Program Files\PuTTY\pscp.exe" -hostkey $hostkey -pw $pw deploy\nginx\tomato.beinai.cc.conf "${remote}:/tmp/tomato-beinai-nginx/deploy/nginx/"
& "C:\Program Files\PuTTY\pscp.exe" -hostkey $hostkey -pw $pw deploy\nginx\install-tomato-beinai-nginx.sh "${remote}:/tmp/tomato-beinai-nginx/deploy/nginx/"
& "C:\Program Files\PuTTY\pscp.exe" -hostkey $hostkey -pw $pw deploy\ssl\tomato.beinai.cc\tomato.beinai.cc_bundle.crt "${remote}:/tmp/tomato-beinai-nginx/deploy/ssl/tomato.beinai.cc/"
& "C:\Program Files\PuTTY\pscp.exe" -hostkey $hostkey -pw $pw deploy\ssl\tomato.beinai.cc\tomato.beinai.cc.key "${remote}:/tmp/tomato-beinai-nginx/deploy/ssl/tomato.beinai.cc/"

& "C:\Program Files\PuTTY\plink.exe" -ssh $remote -hostkey $hostkey -pw $pw -batch `
  "cd /tmp/tomato-beinai-nginx && bash deploy/nginx/install-tomato-beinai-nginx.sh"
```

安装脚本行为：

1. 创建 `/etc/nginx/ssl/tomato.beinai.cc`。
2. 复制 tomato 证书和私钥，私钥权限设为 `600`。
3. 只写入 `/etc/nginx/conf.d/tomato.beinai.cc.conf`。
4. 执行 `nginx -t`。
5. 语法失败时只回滚 tomato 配置，不碰 qbook。
6. 语法通过后执行 `systemctl reload nginx`。

2026-06-19 实际执行成功时备份目录：

```text
/root/nginx-backup-tomato.beinai.cc-20260619-050117
```

2026-06-19 已给服务器 `root` 用户安装本机公钥：

```text
本机私钥: C:/Users/dabin/.ssh/id_rsa
本机公钥: C:/Users/dabin/.ssh/id_rsa.pub
验证命令: ssh -F NUL -o BatchMode=yes -i C:/Users/dabin/.ssh/id_rsa root@47.86.203.13 "hostname && whoami"
```

后续发布优先使用 SSH 私钥登录，不要把服务器密码写进项目文档、全局记忆、脚本或提交。

### 发布后验证

从本机验证 HTTP 自动跳转：

```powershell
curl.exe -I --max-time 20 http://tomato.beinai.cc/
```

期望结果：

```text
HTTP/1.1 301 Moved Permanently
Location: https://tomato.beinai.cc/
```

验证 HTTPS：

```powershell
curl.exe -I --max-time 20 https://tomato.beinai.cc/
curl.exe -sS --max-time 20 https://tomato.beinai.cc/health
```

期望结果：

```text
HTTP/1.1 200 OK
{"status":"ok"}
```

验证证书：

```text
CN = tomato.beinai.cc
SAN = DNS:tomato.beinai.cc
Issuer = TrustAsia DV TLS RSA CA 2024
valid_to = Sep 16 15:59:59 2026 GMT
```

验证 qbook 未受影响：

```powershell
curl.exe -I --max-time 20 https://qbook.top/
```

qbook 的证书应仍为：

```text
CN = qbook.top
SAN = DNS:qbook.top
```

服务器上确认两个站点配置并存：

```bash
nginx -T 2>/dev/null | grep -n tomato.beinai.cc | head -20
nginx -T 2>/dev/null | grep -n qbook.top | head -20
```

2026-06-19 验证结果：

```text
tomato.beinai.cc 配置: /etc/nginx/conf.d/tomato.beinai.cc.conf
qbook.top 配置: /etc/nginx/sites-enabled/qbook.top
http://tomato.beinai.cc/ -> 301 到 https://tomato.beinai.cc/
https://tomato.beinai.cc/ -> 200 OK
https://tomato.beinai.cc/health -> {"status":"ok"}
qbook.top HTTPS 证书仍为 qbook.top
```
