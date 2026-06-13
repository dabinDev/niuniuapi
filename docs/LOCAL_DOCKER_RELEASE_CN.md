# Niuniu API 本地 Docker 构建发布流程

本文档用于发布 `niuniuapi` / `灵犀文创` 这一套独立服务。目标是：**在本机完成 Docker 镜像构建，线上服务器只加载镜像并重启新容器，避免再次卡住原有 Sub2 服务。**

## 服务器与路径

本地项目目录：

```powershell
E:\ForkProject\niuniuapi
```

首尔服务器：

```text
IP: 43.155.249.112
SSH 用户: root
SSH 私钥路径: G:/my-linux/shouer.pem
```

注意：不要把私钥文件内容写进仓库，也不要提交 `.env`、证书私钥、数据库密码。文档只记录私钥路径。

线上新服务目录：

```text
/opt/niuniuapi
/opt/niuniuapi/deploy
/opt/niuniuapi/deploy/.env
/opt/niuniuapi/deploy/docker-compose.niuniuapi.yml
```

线上临时镜像包上传位置：

```text
/tmp/niuniuapi-lingxi.tar
```

Nginx 新站配置：

```text
/opt/nginx/conf.d/lingxi.cylonai.cn.conf
```

## 绝对不要影响旧 Sub2

旧 Sub2 正在同一台服务器运行，发布 `niuniuapi` 时不要操作这些资源：

```text
旧目录: /opt/sub2api
旧应用容器: sub2api
旧数据库容器: sub2api-postgres
旧 Redis 容器: sub2api-redis
旧端口: 18080
旧域名: token.cylonai.cn, sub.cyroute.cn
```

新 `niuniuapi` 必须使用这些独立资源：

```text
新目录: /opt/niuniuapi
新应用容器: niuniuapi
新数据库容器: niuniuapi-postgres
新 Redis 容器: niuniuapi-redis
新端口: 18089
新域名: lingxi.cylonai.cn
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

docker build -t niuniuapi:lingxi .
```

如果本机 Docker 访问 Docker Hub 很慢或失败，可以使用镜像源构建。这个命令只在本机运行，不会碰线上服务器：

```powershell
docker build --progress=plain `
  --build-arg NODE_IMAGE=docker.1ms.run/library/node:24-alpine `
  --build-arg GOLANG_IMAGE=docker.1ms.run/library/golang:1.26.4-alpine `
  --build-arg ALPINE_IMAGE=docker.1ms.run/library/alpine:3.21 `
  --build-arg POSTGRES_IMAGE=docker.1ms.run/library/postgres:18-alpine `
  -t niuniuapi:lingxi .
```

如果需要带版本标签，推荐同时打一个时间戳标签：

```powershell
$tag = "lingxi-" + (Get-Date -Format "yyyyMMddHHmm")
docker build -t "niuniuapi:$tag" -t niuniuapi:lingxi .
```

构建完成后本地验证镜像存在：

```powershell
docker images niuniuapi --format 'table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.CreatedSince}}\t{{.Size}}'
```

如果先构建了测试标签，例如 `niuniuapi:lingxi-local-test`，发布前再改成正式标签：

```powershell
docker tag niuniuapi:lingxi-local-test niuniuapi:lingxi
```

排障建议：

```text
1. 构建卡住时先加 --progress=plain，看具体卡在哪一步。
2. 只检查本机 docker-buildx / Docker Desktop，不要登录服务器处理构建问题。
3. 不要从生产服务器 docker save 或 docker pull 镜像回来，本地构建失败就继续修本地环境。
```

## 导出镜像包

```powershell
docker save niuniuapi:lingxi -o C:\Users\dabin\AppData\Local\Temp\niuniuapi-lingxi.tar
```

确认文件生成：

```powershell
Get-Item C:\Users\dabin\AppData\Local\Temp\niuniuapi-lingxi.tar
```

## 上传镜像到服务器

```powershell
scp -F NUL -i G:/my-linux/shouer.pem `
  C:/Users/dabin/AppData/Local/Temp/niuniuapi-lingxi.tar `
  root@43.155.249.112:/tmp/niuniuapi-lingxi.tar
```

如果只改了 compose 文件，也可以单独上传 compose 到新服务目录，注意路径是 `/opt/niuniuapi`，不是 `/opt/sub2api`：

```powershell
scp -F NUL -i G:/my-linux/shouer.pem `
  E:/ForkProject/niuniuapi/deploy/docker-compose.niuniuapi.yml `
  root@43.155.249.112:/opt/niuniuapi/deploy/docker-compose.niuniuapi.yml
```

## 服务器加载镜像并重启新服务

登录服务器：

```powershell
ssh -F NUL -i G:/my-linux/shouer.pem root@43.155.249.112
```

在服务器执行：

```bash
docker load -i /tmp/niuniuapi-lingxi.tar
docker images niuniuapi

cd /opt/niuniuapi/deploy
docker compose -f docker-compose.niuniuapi.yml --env-file .env up -d --no-build
```

关键点：

```text
必须加 --no-build
```

这样服务器只会使用已经上传的镜像，不会在生产服务器上重新跑 `npm install`、`vite build`、`go build`，避免把旧 Sub2 卡住。

## 发布后验证

检查新旧容器状态：

```bash
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}' | egrep 'NAMES|niuniuapi|sub2api|nginx-token'
```

期望看到：

```text
niuniuapi            Up ... (healthy)   0.0.0.0:18089->8080/tcp
niuniuapi-postgres   Up ... (healthy)   5432/tcp
niuniuapi-redis      Up ...             6379/tcp
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
curl -sS -m 8 -I -H 'Host: lingxi.cylonai.cn' http://127.0.0.1/home | head
```

本机 Windows 也可以模拟 DNS 访问：

```powershell
curl.exe -I --max-time 15 --resolve lingxi.cylonai.cn:80:43.155.249.112 http://lingxi.cylonai.cn/home
```

## DNS 与 SSL

`lingxi.cylonai.cn` 需要在腾讯云 DNS 添加：

```text
主机记录: lingxi
记录类型: A
记录值: 43.155.249.112
```

当前 `18089` 不需要对公网开放。推荐只让 Nginx 通过 80/443 访问新服务：

```text
公网: 80/443
内网/宿主机: 18089
```

HTTPS 需要单独申请 `lingxi.cylonai.cn` 的 SSL 证书。不能直接复用 `token.cylonai.cn` 或 `sub.cyroute.cn` 的证书，否则浏览器会提示证书域名不匹配。

证书放置建议：

```text
/opt/nginx/ssl/lingxi.cylonai.cn_bundle.pem
/opt/nginx/ssl/lingxi.cylonai.cn.key
```

证书配置完成后，再把 `/opt/nginx/conf.d/lingxi.cylonai.cn.conf` 增加 443 server，并执行：

```bash
docker exec nginx-token nginx -t
docker exec nginx-token nginx -s reload
```

## 回滚

如果新镜像启动失败，但旧 Sub2 正常，不要动旧 Sub2。只处理新服务：

```bash
cd /opt/niuniuapi/deploy
docker compose -f docker-compose.niuniuapi.yml --env-file .env logs --tail=200 niuniuapi
docker compose -f docker-compose.niuniuapi.yml --env-file .env restart niuniuapi
```

如果需要回滚到上一版镜像，先确认本机或服务器保留了旧镜像标签，例如：

```bash
docker images niuniuapi
docker tag niuniuapi:lingxi-上一版本 niuniuapi:lingxi
cd /opt/niuniuapi/deploy
docker compose -f docker-compose.niuniuapi.yml --env-file .env up -d --no-build
```

如果只是新站完全不要对外，停新服务即可：

```bash
cd /opt/niuniuapi/deploy
docker compose -f docker-compose.niuniuapi.yml --env-file .env stop niuniuapi
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
niuniuapi    lingxi-local-test   7851fa1c445d   144MB
```

这次没有发布、没有登录服务器、没有从服务器拉取或导出镜像。
