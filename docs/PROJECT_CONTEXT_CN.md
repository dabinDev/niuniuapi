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
灵犀文创
```

定位：

```text
面向小说作者、内容创作者和 AIGC 创作者的 AI 写作与创作工作站。
后续可以销售 token，同时提供 skill、插件和作者工作流能力。
```

首页品牌演进：原 Sub2API → `灵犀文创`（2026-06-13）→ `老番茄`（2026-06-13）→ **`烂番茄`（2026-06-14，当前）**。命名详见「命名与品牌」。

## 命名与品牌（2026-06-14）

当前对外品牌名：

```text
烂番茄（LANFANQIE STUDIO），印章图标 🍅
```

命名策略说明：

```text
- 走「擦边番茄小说」路线借番茄系流量，但番茄/番茄小说/红果均为字节跳动注册商标，
  仅用于宣传文案与 SEO 关键词，不要做主体商标/域名 Logo。
- 「烂番茄」双重碰瓷：番茄小说（字节）+ 烂番茄影评网 Rotten Tomatoes（Fandango 商标）。
  优势是记忆点强、且与「拆书 / 爆款分析」的"内容质检评级"定位天然契合，
  可立「毒舌创作质检台：专挑烂梗/烂节奏，让你的书不烂」人设。
- 风险：「烂」对付费生成工具有"低质"负面暗示，需靠"质检/毒舌"人设翻转语义；
  SEO 会被 Rotten Tomatoes 影评网压制；商标同样只宜用于文案，不宜做主体注册。
- 历史候选：「灵犀文创」（已弃）、「老番茄」（撞 B 站头部 UP 主，已弃）、
  「大番茄中转站」（合作伙伴提议，弃用"中转站"——暴露中转、面向作家不友好、监管敏感）。
- 「毒舌质检」是否作为产品主线尚未最终拍板，决定了「烂」是加分还是减分项。
```

前端品牌已替换文件：

```text
frontend/src/views/HomeView.vue  （brand-seal = 🍅，品牌文案 = 烂番茄 / LANFANQIE STUDIO，番茄红主色 #e8412e）
frontend/index.html              （标签标题 = 烂番茄 · AI 写作与内容创作平台）
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

硬性发布红线（2026-06-19 追加）：

```text
- 任何线上发布前，必须先在本机完成 `docker build -t niuniuapi:lingxi .` 或等价镜像源构建，并确认构建成功。
- 生产服务器只允许 `docker load` 已上传镜像，再用 `docker compose ... up -d --no-build` 重启新 niuniuapi 服务。
- 禁止在生产服务器完整执行前端 npm/pnpm 安装、vite/vue-tsc 构建或 Go 编译；这会抢占 CPU/IO，把同机旧 Sub2 服务卡慢。
- 发布、回滚、排障都只操作 `/opt/niuniuapi`、`18089`、`niuniuapi*` 资源；不要操作 `/opt/sub2api`、`18080`、`sub2api*` 旧服务资源。
```

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
