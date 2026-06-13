# 灵犀文创 / Niuniu API 项目上下文

本文档用于把当前 Codex 对话里的关键项目上下文沉淀到仓库中，方便后续打开项目后直接检索。

## 项目地址

本地项目目录：

```text
E:\ForkProject\niuniuapi
```

Git 仓库：

```text
git@github.com:dabinDev/niuniuapi.git
```

当前开发分支：

```text
writer-workbench-v0
```

PR 创建地址：

```text
https://github.com/dabinDev/niuniuapi/pull/new/writer-workbench-v0
```

本地开发访问地址：

```text
http://127.0.0.1:5177/home
```

## 产品方向

项目名称：

```text
灵犀文创
```

定位：

```text
面向小说作者、内容创作者和 AIGC 创作者的 AI 写作与创作工作站。
后续可以销售 token，同时提供 skill、插件和作者工作流能力。
```

当前首页已经从原 Sub2API 风格改成 `灵犀文创` 创作者平台首页。

## 已完成的重要改动

1. 重做首页：

```text
frontend/src/views/HomeView.vue
```

2. 增加 GSAP 动效依赖：

```text
frontend/package.json
frontend/pnpm-lock.yaml
```

3. 新增独立部署 compose，不与旧 Sub2 共用数据库、Redis 或端口：

```text
deploy/docker-compose.niuniuapi.yml
```

4. 修复 Docker 构建：

```text
Dockerfile
.dockerignore
```

5. 新增本地 Docker 构建与发布文档：

```text
docs/LOCAL_DOCKER_RELEASE_CN.md
```

## 部署隔离原则

旧 Sub2 服务不要动：

```text
旧目录: /opt/sub2api
旧应用容器: sub2api
旧数据库容器: sub2api-postgres
旧 Redis 容器: sub2api-redis
旧端口: 18080
旧域名: token.cylonai.cn, sub.cyroute.cn
```

新灵犀文创服务使用独立资源：

```text
新目录: /opt/niuniuapi
新应用容器: niuniuapi
新数据库容器: niuniuapi-postgres
新 Redis 容器: niuniuapi-redis
新端口: 18089
新域名: lingxi.cylonai.cn
```

## 发布注意事项

发布时优先使用本机 Docker 构建镜像，再上传镜像包到服务器。

不要在生产服务器上执行完整构建，因为服务器资源较小，之前远程构建会让旧 Sub2 访问变慢。

详细步骤见：

```text
docs/LOCAL_DOCKER_RELEASE_CN.md
```

## 最近验证记录

2026-06-13 已在本机 Docker 完成一次镜像构建验证：

```text
REPOSITORY   TAG                 IMAGE ID       SIZE
niuniuapi    lingxi-local-test   7851fa1c445d   144MB
```

本次验证没有发布、没有登录服务器、没有从服务器拉取或导出镜像。

## 最近提交

```text
c2fdb7a8 docs: document local docker build mirrors
9e3c4231 docs: add local docker release guide
2cd731fb feat: launch lingxi creative homepage
```

