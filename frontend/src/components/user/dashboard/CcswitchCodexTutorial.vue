<template>
  <section id="ccswitch-codex-tutorial" class="tutorial-entry overflow-hidden rounded-[1.75rem] border border-rose-100 bg-gradient-to-br from-[#fff7ed] via-white to-[#fef2f2] shadow-sm dark:border-rose-500/20 dark:from-[#21130f] dark:via-dark-900 dark:to-[#261416]">
    <div class="relative p-5 sm:p-6 lg:p-7">
      <div class="pointer-events-none absolute right-0 top-0 h-40 w-40 rounded-full bg-rose-300/20 blur-3xl dark:bg-rose-500/10"></div>
      <div class="relative grid gap-5 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center">
        <div class="max-w-3xl">
          <div class="inline-flex items-center gap-2 rounded-full bg-rose-100 px-3 py-1 text-xs font-semibold uppercase tracking-[0.22em] text-rose-700 dark:bg-rose-500/10 dark:text-rose-200">
            CCSWITCH + Codex
          </div>
          <h2 class="mt-4 text-2xl font-black tracking-tight text-gray-950 dark:text-white">
            先看完整教程，再接入本地工具
          </h2>
          <p class="mt-3 text-sm leading-7 text-gray-600 dark:text-dark-300">
            教程页会按步骤说明如何创建番茄 API 密钥、一键导入 CCSWITCH、配置 Codex 客户端与 Codex CLI，
            并把截图插入到对应步骤里。遇到模型列表 401、余额不足、Base URL 写错时，也可以直接按教程排查。
          </p>
          <div class="mt-4 flex flex-wrap gap-2 text-xs font-semibold text-rose-700 dark:text-rose-200">
            <span class="rounded-full bg-white/80 px-3 py-1 dark:bg-dark-800">一键导入 CCSWITCH</span>
            <span class="rounded-full bg-white/80 px-3 py-1 dark:bg-dark-800">Codex 客户端</span>
            <span class="rounded-full bg-white/80 px-3 py-1 dark:bg-dark-800">Codex CLI</span>
            <span class="rounded-full bg-white/80 px-3 py-1 dark:bg-dark-800">常见错误排查</span>
          </div>
        </div>

        <div class="flex flex-col gap-3 sm:flex-row lg:flex-col">
          <RouterLink
            to="/tutorials/ccswitch-codex"
            class="inline-flex shrink-0 items-center justify-center rounded-2xl bg-gray-950 px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-rose-900/10 transition hover:-translate-y-0.5 hover:bg-rose-700 dark:bg-white dark:text-gray-950 dark:hover:bg-rose-100"
          >
            查看完整教程
          </RouterLink>
          <RouterLink
            to="/keys"
            class="inline-flex shrink-0 items-center justify-center rounded-2xl border border-rose-200 bg-white/78 px-4 py-2.5 text-sm font-semibold text-rose-700 transition hover:-translate-y-0.5 hover:border-rose-300 hover:bg-white dark:border-rose-500/20 dark:bg-dark-800/70 dark:text-rose-200"
          >
            去 API 密钥页
          </RouterLink>
        </div>
      </div>
    </div>
  </section>
</template>
