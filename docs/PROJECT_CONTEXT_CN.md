# 番茄 / Tomato 项目上下文

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
http://127.0.0.1:5180/home
```

⚠️ 本地端口坑：本机的 `cylonai-app`（赛隆 AI / Next.js）常驻占用 `:3000`（生产构建）和 `:3001`（turbopack dev）。
前端 vite 默认端口会与之冲突，导致打开后看到的是赛隆 AI 页面而非本项目。启动 vite 时务必指定专用端口：

```text
cd frontend; pnpm dev --port 5180 --strictPort --host 127.0.0.1
```

## 产品方向

项目名称：

```text
番茄
```

定位：

```text
面向小说作者、内容创作者和 AIGC 创作者的 AI 写作与创作工作站。
后续可以销售 token，同时提供 skill、插件和作者工作流能力。
```

首页品牌演进：原 Sub2API → 早期创作站命名 → `老番茄`（2026-06-13，已弃用）→ **`番茄`（当前）**。命名详见「命名与品牌」。

## 命名与品牌（2026-06-14）

当前对外品牌名：

```text
番茄（TOMATO STUDIO），印章图标 🍅
```

命名策略说明：

```text
- 走「擦边番茄小说」路线借番茄系流量，但番茄/番茄小说/红果均为字节跳动注册商标，
  仅用于宣传文案与 SEO 关键词，不要做主体商标/域名 Logo。
- 「番茄」借番茄小说相关流量做内容方向联想。
  优势是记忆点强、且与「拆书 / 爆款分析」的"内容质检评级"定位天然契合，
  可立「毒舌创作质检台：专挑烂梗/烂节奏，让你的书不烂」人设。
- 风险：番茄/番茄小说/红果均为他人商标，商标和域名主体要继续避险；
  SEO 会与番茄小说相关内容竞争，不能把第三方平台名做主体商标承诺。
- 历史候选：早期创作站命名（已弃）、「老番茄」（撞 B 站头部 UP 主，已弃）、
  「大番茄中转站」（合作伙伴提议，弃用"中转站"——暴露中转、面向作家不友好、监管敏感）。
- 「毒舌质检」可作为产品语气，但当前项目/部署统一使用「番茄 / tomato」。
```

前端品牌已替换文件：

```text
frontend/src/views/HomeView.vue  （brand-seal = 🍅，品牌文案 = 番茄 / TOMATO STUDIO，番茄红主色 #e8412e）
frontend/index.html              （标签标题 = 番茄 · AI 写作与内容创作平台）
```

## 功能矩阵（规划）

> 完整功能描述书见 [`docs/PRODUCT_SPEC_CN.md`](./PRODUCT_SPEC_CN.md)。核心 P0 = 拆书&爆款分析 / 封面生成 / 剧本生成；
> Sub2 旧功能收纳：用户侧并入「Token Club」Tab，管理员侧并入控制台「系统管理」Tab（仅管理员可见）；变现 = Token 按量 + 套餐订阅。

面向小说作者/写手的一站式工作台，规划功能：

```text
- 小说封面生成（AI 封面）
- 小说拆书 / 拆解分析
- 番茄小说下载器（注意：抓取番茄内容有 ToS/版权风险，建议定位为个人备份工具，不做主打卖点）
- 爆款分析
- 剧本生成
- token 套餐（变现核心，底层复用 sub2api 的额度分发与支付）
```

## 域名查询结果（2026-06-13，DoH/RDAP 实测）

```text
laofanqie.com     已注册（阿里云/万网 hichina NS，2017 年起持有）
laofanqie.net     已注册（同 hichina NS）
laofanqie.cn      已注册（同 hichina NS）
laofanqie.com.cn  NXDOMAIN，疑似可注册
laofanqie.app     NXDOMAIN，疑似可注册
```

结论：

```text
laofanqie 的 .com/.net/.cn 三大主力后缀均由同一阿里云账户持有，
高度疑似 B 站「老番茄」品牌方或其 MCN 防御性注册，主力域名基本拿不到。
可考虑的方向：laofanqie.com.cn / laofanqie.app，或换更安全的主体名
（如「老番茄创作站」对应的其它拼音域名）。注册前到注册商控制台二次确认。
```

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
deploy/docker-compose.tomato.yml
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

新番茄服务使用独立资源：

```text
新目录: /opt/niuniuapi
新应用容器: tomato
新数据库容器: tomato-postgres
新 Redis 容器: tomato-redis
新端口: 18089
新域名: tomato.beinai.cc
```

## 发布注意事项

发布时优先使用本机 Docker 构建镜像，再上传镜像包到服务器。

不要在生产服务器上执行完整构建，因为服务器资源较小，之前远程构建会让旧 Sub2 访问变慢。

硬性发布红线（2026-06-19 追加）：

```text
- 任何线上发布前，必须先在本机完成 `docker build -t tomato:latest .` 或等价镜像源构建，并确认构建成功。
- 生产服务器只允许 `docker load` 已上传镜像，再用 `docker compose ... up -d --no-build` 重启 tomato 服务。
- 禁止在生产服务器完整执行前端 npm/pnpm 安装、vite/vue-tsc 构建或 Go 编译；这会抢占 CPU/IO，把同机旧 Sub2 服务卡慢。
- 发布、回滚、排障都只操作 `/opt/niuniuapi`、`18089`、`tomato*` 资源；不要操作 `/opt/sub2api`、`18080`、`sub2api*` 旧服务资源。
```

重启 Codex 后也必须继续遵守的发布记忆（2026-06-19 再次确认）：

```text
- 任何时候都不要为了省事直接在生产服务器构建或发布新代码；这会抢占 CPU/IO，可能把同机旧 Sub2 服务卡慢。
- 线上发布的唯一允许路径是：先在本机 `E:\ForkProject\niuniuapi` 完成 Docker 镜像构建并确认成功，再 `docker save` 上传镜像包到服务器。
- 服务器只允许 `docker load` 已上传镜像，并在 `/opt/niuniuapi/deploy` 使用 `docker compose ... up -d --no-build` 重启新服务。
- 如果本机 Docker Desktop 无法启动或 `docker build` 失败，就停止发布流程，只继续本地代码/前端/测试修复；不要登录服务器绕过本地构建门禁。
- 旧 Sub2 服务是隔离保护对象：不要重启、down、删除、覆盖或迁移 `/opt/sub2api`、`sub2api*` 容器、`18080` 端口相关资源。
```
详细步骤见：

```text
docs/LOCAL_DOCKER_RELEASE_CN.md
```

## 最近验证记录

2026-06-13 已在本机 Docker 完成一次镜像构建验证：

```text
REPOSITORY   TAG                 IMAGE ID       SIZE
tomato       tomato-local-test   7851fa1c445d   144MB
```

本次验证没有发布、没有登录服务器、没有从服务器拉取或导出镜像。

## 最近提交

```text
c2fdb7a8 docs: document local docker build mirrors
9e3c4231 docs: add local docker release guide
2cd731fb feat: launch tomato creative homepage
```
