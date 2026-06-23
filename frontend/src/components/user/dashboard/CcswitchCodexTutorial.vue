<template>
  <section id="ccswitch-codex-tutorial" class="ccswitch-tutorial overflow-hidden rounded-[1.75rem] border border-rose-100 bg-gradient-to-br from-[#fff7ed] via-white to-[#fef2f2] shadow-sm dark:border-rose-500/20 dark:from-[#21130f] dark:via-dark-900 dark:to-[#261416]">
    <div class="relative p-5 sm:p-6 lg:p-7">
      <div class="pointer-events-none absolute right-0 top-0 h-40 w-40 rounded-full bg-rose-300/20 blur-3xl dark:bg-rose-500/10"></div>
      <div class="relative flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
        <div class="max-w-3xl">
          <div class="inline-flex items-center gap-2 rounded-full bg-rose-100 px-3 py-1 text-xs font-semibold uppercase tracking-[0.22em] text-rose-700 dark:bg-rose-500/10 dark:text-rose-200">
            CCSWITCH + Codex
          </div>
          <h2 class="mt-4 text-2xl font-black tracking-tight text-gray-950 dark:text-white">
            先完成接入，再开始写作和调用模型
          </h2>
          <p class="mt-3 text-sm leading-7 text-gray-600 dark:text-dark-300">
            下面的教程基于本系统真实页面截图：你可以用 CCSWITCH 一键导入，也可以手动把密钥写入本地 Codex 配置。Codex 客户端和 Codex CLI 都能使用同一套
            <code class="rounded bg-white/80 px-1.5 py-0.5 text-rose-700 dark:bg-dark-800 dark:text-rose-200">config.toml</code>
            与
            <code class="rounded bg-white/80 px-1.5 py-0.5 text-rose-700 dark:bg-dark-800 dark:text-rose-200">auth.json</code>
            配置。
          </p>
        </div>
        <RouterLink
          to="/keys"
          class="inline-flex shrink-0 items-center justify-center rounded-2xl bg-gray-950 px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-rose-900/10 transition hover:-translate-y-0.5 hover:bg-rose-700 dark:bg-white dark:text-gray-950 dark:hover:bg-rose-100"
        >
          去密钥页操作
        </RouterLink>
      </div>

      <div class="relative mt-6 grid gap-4 lg:grid-cols-2">
        <article
          v-for="step in steps"
          :key="step.title"
          class="rounded-3xl border border-white/80 bg-white/82 p-4 shadow-sm backdrop-blur dark:border-white/10 dark:bg-dark-800/70"
        >
          <div class="flex items-start gap-3">
            <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-2xl bg-rose-600 text-sm font-black text-white">
              {{ step.index }}
            </span>
            <div>
              <h3 class="text-base font-bold text-gray-950 dark:text-white">{{ step.title }}</h3>
              <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ step.description }}</p>
            </div>
          </div>
          <ol class="mt-4 space-y-2 text-sm leading-6 text-gray-700 dark:text-dark-200">
            <li v-for="item in step.items" :key="item" class="flex gap-2">
              <span class="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-rose-500"></span>
              <span>{{ item }}</span>
            </li>
          </ol>
          <figure class="mt-4 overflow-hidden rounded-2xl border border-gray-100 bg-gray-50 dark:border-dark-700 dark:bg-dark-900">
            <img :src="step.image" :alt="`${step.title} 真实截图`" class="h-auto w-full" loading="lazy" />
            <figcaption class="border-t border-gray-100 px-3 py-2 text-xs text-gray-500 dark:border-dark-700 dark:text-dark-400">
              {{ step.caption }}
            </figcaption>
          </figure>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
const steps = [
  {
    index: '01',
    title: 'CCSWITCH 一键导入',
    description: '适合已经安装 CCSWITCH 的用户，点击系统里的导入按钮即可把当前 API Key 写入 CCSWITCH。',
    items: [
      '进入“API 密钥”页面，确认密钥已分配 OpenAI 分组并处于启用状态。',
      '点击密钥行右侧的“导入到 CCS”按钮，浏览器会打开 ccswitch:// 导入链接。',
      '在 CCSWITCH 中确认导入的 Base URL、API Key、模型分组，然后保存并切换到该配置。',
    ],
    image: '/tutorial/ccswitch/01-keys-import-ccswitch.png',
    caption: '密钥列表中的“导入到 CCS”按钮会使用当前系统真实密钥生成导入链接。',
  },
  {
    index: '02',
    title: 'Codex 客户端',
    description: '适合使用 Codex 桌面/客户端的用户，先在系统中复制配置，再放入本机 .codex 目录。',
    items: [
      '在密钥行点击“使用密钥”，切换到 Codex CLI 或 Codex CLI (WebSocket) 标签。',
      'Windows 打开 %userprofile%\\.codex，macOS/Linux 打开 ~/.codex；没有目录就新建。',
      '复制弹窗里的 config.toml 和 auth.json；Codex 客户端重启后会读取这两个文件。',
    ],
    image: '/tutorial/ccswitch/02-use-key-codex-client.png',
    caption: '“使用密钥”弹窗会按当前 Base URL 和 API Key 生成 Codex 可用配置。',
  },
  {
    index: '03',
    title: 'Codex CLI',
    description: '适合命令行用户，配置完成后直接在终端运行 codex，即可走本系统网关。',
    items: [
      '安装 Codex CLI 后，确保命令行能执行 codex --version。',
      '把系统生成的 config.toml 与 auth.json 放到 ~/.codex 或 %userprofile%\\.codex。',
      '重新打开终端，运行 codex；如果需要 WebSocket，使用弹窗里的 Codex CLI (WebSocket) 配置。',
    ],
    image: '/tutorial/ccswitch/03-codex-cli-config.png',
    caption: 'Codex CLI 与 Codex 客户端共用本地配置目录，区别只是启动方式不同。',
  },
  {
    index: '04',
    title: '手动配置 config.toml',
    description: '适合不使用一键导入、希望自己维护配置文件的用户。',
    items: [
      'config.toml 中确认 model_provider、model、base_url、wire_api 等字段来自本系统弹窗。',
      'auth.json 中写入 OPENAI_API_KEY，值为你自己的系统 API Key，不要写入上游账号密钥。',
      '保存后重启 Codex；如果请求失败，优先检查密钥分组、余额、模型名和 Base URL。',
    ],
    image: '/tutorial/ccswitch/04-manual-config-files.png',
    caption: '手动方式本质上就是把系统弹窗生成的两段配置准确落到本地文件。',
  },
]
</script>
