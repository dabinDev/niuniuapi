<template>
  <AppLayout>
    <article class="tutorial-page" aria-labelledby="tutorial-title">
      <header class="tutorial-hero">
        <div class="hero-copy">
          <span class="eyebrow">Tomato Access Guide</span>
          <h1 id="tutorial-title">CCSWITCH / Codex 使用教程</h1>
          <p>
            这是一份给番茄用户准备的完整接入指南：先在系统里创建 API 密钥，再选择一键导入 CCSWITCH、
            配置 Codex 客户端、配置 Codex CLI，或手动维护本地配置文件。截图会插入在对应步骤里，
            方便你边看边操作。
          </p>
          <div class="hero-actions">
            <RouterLink to="/keys">去 API 密钥页</RouterLink>
            <RouterLink to="/dashboard" class="secondary">返回创作台</RouterLink>
          </div>
        </div>

        <aside class="hero-note" aria-label="教程适用场景">
          <span>适用场景</span>
          <strong>把番茄里的密钥接入本地 AI 工具</strong>
          <p>
            如果你遇到模型列表 401、余额不足、Base URL 写错、Codex 不读配置等问题，
            本页后半部分也整理了排查顺序。
          </p>
        </aside>
      </header>

      <section class="prep-card" aria-labelledby="prep-title">
        <div>
          <span class="section-mark">开始前先准备</span>
          <h2 id="prep-title">确认密钥、分组、余额和工具环境</h2>
        </div>
        <ol>
          <li>进入番茄的“API 密钥”页面，创建或选择一把你要给 Codex 使用的 API Key。</li>
          <li>确认这把密钥已经绑定可用分组，分组内包含你要使用的对话模型和生图模型。</li>
          <li>确认账户余额充足；如果余额不足，模型调用和封面生成都会失败。</li>
          <li>如果要用 CCSWITCH，先在本机安装并确认浏览器能打开 <code>ccswitch://</code> 链接。</li>
          <li>如果要用 Codex 客户端或 Codex CLI，先确认本机存在或可以新建 <code>.codex</code> 配置目录。</li>
        </ol>
      </section>

      <nav class="step-index" aria-label="教程目录">
        <a v-for="step in steps" :key="step.id" :href="`#${step.id}`">
          <span>{{ step.index }}</span>
          {{ step.shortTitle }}
        </a>
      </nav>

      <section class="steps">
        <section v-for="step in steps" :id="step.id" :key="step.id" class="tutorial-step">
          <div class="step-heading">
            <span>{{ step.index }}</span>
            <div>
              <h2>{{ step.title }}</h2>
              <p>{{ step.summary }}</p>
            </div>
          </div>

          <div class="step-body">
            <div class="step-copy">
              <h3>{{ step.operationTitle }}</h3>
              <ol>
                <li v-for="item in step.items" :key="item" v-html="item"></li>
              </ol>

              <div v-if="step.command" class="code-card">
                <span>{{ step.command.label }}</span>
                <pre><code>{{ step.command.value }}</code></pre>
              </div>

              <div class="tip-box">
                <strong>{{ step.tipTitle }}</strong>
                <p>{{ step.tip }}</p>
              </div>
            </div>

            <figure class="tutorial-shot">
              <img :src="step.image" :alt="`${step.title} 教程截图`" loading="lazy" />
              <figcaption>{{ step.caption }}</figcaption>
            </figure>
          </div>
        </section>
      </section>

      <section class="troubleshooting" aria-labelledby="troubleshooting-title">
        <div class="trouble-heading">
          <span class="section-mark">常见问题排查</span>
          <h2 id="troubleshooting-title">先看错误文字，再按链路排查</h2>
          <p>这几个问题最容易发生在第一次接入时。优先按下面顺序检查，通常比反复重装工具更快。</p>
        </div>

        <div class="trouble-grid">
          <article v-for="item in troubleshooting" :key="item.title">
            <span>{{ item.badge }}</span>
            <h3>{{ item.title }}</h3>
            <p>{{ item.description }}</p>
            <ol>
              <li v-for="check in item.checks" :key="check">{{ check }}</li>
            </ol>
          </article>
        </div>
      </section>
    </article>
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'

const steps = [
  {
    id: 'create-key',
    index: '01',
    shortTitle: '创建密钥',
    title: '第一步：在番茄创建并确认 API 密钥',
    summary: '所有接入方式都从同一把番茄 API Key 开始，先确认密钥启用、分组正确、余额充足。',
    operationTitle: '你需要这样做',
    items: [
      '打开 <strong>API 密钥</strong> 页面，选择已有密钥，或创建一把新的密钥。',
      '确认密钥状态为启用，并且已绑定包含对话模型的 OpenAI 分组。',
      '如果要生成小说封面，还要确认系统已经自动选中最新生图模型，或你已经手动测试并保存过生图模型。',
    ],
    image: '/tutorial/ccswitch/01-keys-import-ccswitch.png',
    caption: '密钥列表中的“导入到 CCS”入口会使用当前番茄密钥生成导入链接；这一步也能确认密钥是否可用。',
    tipTitle: '先别急着复制',
    tip: '第一次配置时，先看密钥所属分组和余额。密钥本身没问题，但分组没有模型或账户没余额，也会导致后续工具请求失败。',
  },
  {
    id: 'import-ccswitch',
    index: '02',
    shortTitle: '导入 CCSWITCH',
    title: '第二步：一键导入到 CCSWITCH',
    summary: '适合已经安装 CCSWITCH 的用户，点击按钮后让本机 CCSWITCH 接管 Base URL、API Key 和模型配置。',
    operationTitle: '一键导入流程',
    items: [
      '在密钥行右侧点击 <strong>导入到 CCS</strong>，浏览器会打开 <code>ccswitch://</code> 协议链接。',
      '在 CCSWITCH 弹出的导入确认页里检查 Base URL、API Key、模型分组是否来自番茄。',
      '保存并切换到这套配置后，再回到 Codex 或其他工具里选择 CCSWITCH 提供的模型入口。',
    ],
    image: '/tutorial/ccswitch/02-use-key-codex-client.png',
    caption: '如果浏览器没有拉起 CCSWITCH，通常是本机没有安装、协议没有注册，或浏览器拦截了外部应用跳转。',
    tipTitle: '导入后仍要切换配置',
    tip: 'CCSWITCH 可以保存多套来源。导入成功不等于当前正在使用，记得在 CCSWITCH 里把番茄这套配置切为当前配置。',
  },
  {
    id: 'codex-client-cli',
    index: '03',
    shortTitle: '配置 Codex',
    title: '第三步：配置 Codex 客户端或 Codex CLI',
    summary: 'Codex 客户端和 Codex CLI 可以共用同一套本地配置，只要把系统生成的文件放进 .codex 目录。',
    operationTitle: '复制系统生成的配置',
    items: [
      '在密钥行点击 <strong>使用密钥</strong>，切换到 Codex CLI 或 Codex CLI (WebSocket) 标签。',
      '复制弹窗里的 <code>config.toml</code> 和 <code>auth.json</code> 内容。',
      'Windows 放到 <code>%userprofile%\\.codex</code>，macOS/Linux 放到 <code>~/.codex</code>。',
      '重启 Codex 客户端，或重新打开终端运行 <code>codex</code>，让工具重新读取配置。',
    ],
    command: {
      label: 'Windows 配置目录',
      value: '%userprofile%\\.codex',
    },
    image: '/tutorial/ccswitch/03-codex-cli-config.png',
    caption: 'Codex 客户端和 CLI 的核心差别是启动方式；配置文件落点一致，都是本机 .codex 目录。',
    tipTitle: '不要混用上游密钥',
    tip: 'auth.json 中的 OPENAI_API_KEY 应该写番茄生成的 API Key，不要把上游平台账号密钥直接写进本地 Codex 配置。',
  },
  {
    id: 'manual-config',
    index: '04',
    shortTitle: '手动配置',
    title: '第四步：手动写入 config.toml 与 auth.json',
    summary: '适合不用一键导入、希望自己维护文件的人。关键是 Base URL、model_provider 和 API Key 必须对应同一套番茄配置。',
    operationTitle: '手动落文件时检查这些字段',
    items: [
      '<code>config.toml</code> 里检查 <code>model_provider</code>、<code>model</code>、<code>base_url</code>、<code>wire_api</code>。',
      '<code>auth.json</code> 里检查 <code>OPENAI_API_KEY</code>，值必须是番茄 API 密钥。',
      '保存文件后完全退出并重启 Codex；如果只刷新页面，旧配置可能还在内存里。',
    ],
    command: {
      label: '常见文件名',
      value: 'config.toml\nauth.json',
    },
    image: '/tutorial/ccswitch/04-manual-config-files.png',
    caption: '手动配置不复杂，本质就是把番茄弹窗生成的两段配置准确写入本地两个文件。',
    tipTitle: '路径和文件名要完全一致',
    tip: '最常见的低级错误是目录写成 .codex 之外的位置，或文件名多了 .txt 后缀。Windows 资源管理器默认可能隐藏扩展名，建议打开扩展名显示。',
  },
]

const troubleshooting = [
  {
    badge: '401',
    title: '获取模型列表失败：Upstream model list request failed with HTTP 401',
    description: '这代表番茄向上游拉取模型列表时鉴权失败，通常不是 Codex 客户端的问题。',
    checks: [
      '管理员后台检查 OpenAI 账号密钥是否过期、填错或被上游禁用。',
      '检查用户 API Key 是否绑定到了正确分组，且分组里存在可用账号。',
      '重新获取模型列表前，先确认 Base URL 指向番茄，而不是误写成上游地址。',
    ],
  },
  {
    badge: 'Balance',
    title: 'Insufficient account balance',
    description: '余额不足会直接影响对话和生图，尤其是小说封面生成时更明显。',
    checks: [
      '先确认番茄用户余额是否足够。',
      '如果用户余额正常，再检查上游生图账号余额或额度是否不足。',
      '管理员可在账号列表里确认当前生图模型走的是哪一个上游账号。',
    ],
  },
  {
    badge: 'Base URL',
    title: '请求到了错误的地址',
    description: 'Codex 能启动但请求失败，常见原因是 config.toml 仍然指向旧网关或别的项目。',
    checks: [
      '重新从番茄“使用密钥”弹窗复制 config.toml。',
      '确认本地只保留一份正在被 Codex 读取的 .codex 配置。',
      '修改后完整重启 Codex 客户端或重新打开终端。',
    ],
  },
]
</script>

<style scoped>
.tutorial-page {
  width: min(100%, 1180px);
  margin: 0 auto;
  padding-bottom: 4rem;
  color: #241815;
  font-family: "Noto Sans SC", "Microsoft YaHei", "PingFang SC", sans-serif;
}

.tutorial-hero {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(18rem, 0.38fr);
  gap: 1rem;
  overflow: hidden;
  border: 1px solid rgba(232, 65, 46, 0.18);
  border-radius: 28px;
  padding: clamp(1.25rem, 4vw, 2rem);
  background:
    radial-gradient(circle at 86% 8%, rgba(232, 65, 46, 0.22), transparent 18rem),
    linear-gradient(135deg, #fff8ef 0%, #fff 52%, #fff2ed 100%);
  box-shadow: 0 24px 80px rgba(74, 35, 22, 0.1);
}

.eyebrow,
.section-mark {
  display: inline-flex;
  width: fit-content;
  border: 1px solid rgba(232, 65, 46, 0.24);
  border-radius: 999px;
  padding: 0.24rem 0.68rem;
  color: #c8351f;
  font-size: 0.72rem;
  font-weight: 950;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.hero-copy h1 {
  margin-top: 0.8rem;
  max-width: 48rem;
  color: #201714;
  font-size: clamp(2.3rem, 7vw, 5.3rem);
  font-weight: 950;
  letter-spacing: -0.065em;
  line-height: 0.94;
}

.hero-copy p {
  margin-top: 1rem;
  max-width: 48rem;
  color: #6c5a52;
  font-size: 1rem;
  line-height: 1.9;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-top: 1.2rem;
}

.hero-actions a {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: #241a16;
  padding: 0.78rem 1.08rem;
  color: #fff7ed;
  font-weight: 950;
  box-shadow: 0 14px 32px rgba(36, 26, 22, 0.18);
}

.hero-actions .secondary {
  border: 1px solid rgba(36, 26, 22, 0.16);
  background: rgba(255, 255, 255, 0.76);
  color: #241a16;
  box-shadow: none;
}

.hero-note {
  align-self: stretch;
  border: 1px solid rgba(36, 26, 22, 0.1);
  border-radius: 22px;
  padding: 1rem;
  background:
    radial-gradient(circle at 86% 22%, rgba(255, 199, 135, 0.2), transparent 9rem),
    #241a16;
  color: #fff7ed;
}

.hero-note span {
  display: inline-flex;
  border-radius: 999px;
  padding: 0.22rem 0.6rem;
  background: rgba(255, 255, 255, 0.1);
  color: #ffc8a5;
  font-size: 0.76rem;
  font-weight: 950;
}

.hero-note strong {
  display: block;
  margin-top: 0.8rem;
  font-size: 1.4rem;
  font-weight: 950;
  line-height: 1.25;
}

.hero-note p {
  margin-top: 0.65rem;
  color: #f0c7bd;
  line-height: 1.75;
}

.prep-card,
.troubleshooting,
.tutorial-step {
  margin-top: 1rem;
  border: 1px solid #eaded8;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.94);
  box-shadow: 0 18px 52px rgba(54, 32, 24, 0.07);
}

.prep-card {
  display: grid;
  grid-template-columns: minmax(14rem, 0.32fr) minmax(0, 1fr);
  gap: 1rem;
  padding: 1.25rem;
}

.prep-card h2,
.trouble-heading h2,
.tutorial-step h2 {
  margin-top: 0.55rem;
  color: #201714;
  font-size: clamp(1.35rem, 3vw, 2.2rem);
  font-weight: 950;
  letter-spacing: -0.04em;
}

.prep-card ol,
.step-copy ol,
.trouble-grid ol {
  display: grid;
  gap: 0.58rem;
  margin: 0;
  padding-left: 1.2rem;
  color: #5f4d46;
  line-height: 1.75;
}

.prep-card code,
.step-copy :deep(code) {
  border-radius: 0.42rem;
  background: #fff1e8;
  padding: 0.08rem 0.34rem;
  color: #b73521;
  font-weight: 800;
}

.step-index {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.7rem;
  margin-top: 1rem;
}

.step-index a {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  border: 1px solid #eaded8;
  border-radius: 16px;
  background: #fffaf6;
  padding: 0.8rem;
  color: #4b3932;
  font-weight: 900;
  transition: transform 0.16s ease, border-color 0.16s ease, box-shadow 0.16s ease;
}

.step-index a:hover {
  transform: translateY(-2px);
  border-color: rgba(232, 65, 46, 0.36);
  box-shadow: 0 14px 32px rgba(54, 32, 24, 0.1);
}

.step-index span,
.step-heading > span {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  background: #e8412e;
  color: #fff;
  font-weight: 950;
}

.step-index span {
  width: 2.4rem;
  height: 2.4rem;
  flex: 0 0 auto;
}

.tutorial-step {
  overflow: hidden;
}

.step-heading {
  display: flex;
  gap: 0.9rem;
  padding: 1.25rem 1.25rem 0;
}

.step-heading > span {
  width: 3rem;
  height: 3rem;
  flex: 0 0 auto;
  box-shadow: 0 14px 30px rgba(232, 65, 46, 0.22);
}

.step-heading p,
.trouble-heading p {
  margin-top: 0.45rem;
  max-width: 50rem;
  color: #6c5a52;
  line-height: 1.75;
}

.step-body {
  display: grid;
  grid-template-columns: minmax(0, 0.92fr) minmax(18rem, 1.08fr);
  gap: 1rem;
  align-items: start;
  padding: 1.25rem;
}

.step-copy {
  display: grid;
  gap: 0.9rem;
}

.step-copy h3 {
  color: #201714;
  font-size: 1rem;
  font-weight: 950;
}

.tip-box,
.code-card {
  border-radius: 16px;
  padding: 0.9rem;
}

.tip-box {
  border: 1px solid rgba(232, 65, 46, 0.18);
  background: #fff7ef;
}

.tip-box strong {
  color: #a8321f;
  font-weight: 950;
}

.tip-box p {
  margin-top: 0.3rem;
  color: #71564d;
  line-height: 1.7;
}

.code-card {
  overflow: hidden;
  border: 1px solid rgba(36, 26, 22, 0.1);
  background: #241a16;
  color: #fff7ed;
}

.code-card span {
  display: block;
  color: #ffc8a5;
  font-size: 0.74rem;
  font-weight: 950;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.code-card pre {
  margin: 0.55rem 0 0;
  overflow-x: auto;
  font-size: 0.92rem;
  line-height: 1.65;
  white-space: pre-wrap;
}

.tutorial-shot {
  overflow: hidden;
  border: 1px solid #eaded8;
  border-radius: 20px;
  background: #fffaf6;
  box-shadow: 0 20px 56px rgba(54, 32, 24, 0.1);
}

.tutorial-shot img {
  display: block;
  width: 100%;
  height: auto;
}

.tutorial-shot figcaption {
  border-top: 1px solid #eaded8;
  padding: 0.78rem 0.9rem;
  color: #71564d;
  font-size: 0.85rem;
  line-height: 1.65;
}

.troubleshooting {
  padding: 1.25rem;
}

.trouble-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.85rem;
  margin-top: 1rem;
}

.trouble-grid article {
  border: 1px solid #eaded8;
  border-radius: 18px;
  background:
    radial-gradient(circle at 100% 0%, rgba(232, 65, 46, 0.11), transparent 8rem),
    #fffaf6;
  padding: 1rem;
}

.trouble-grid span {
  display: inline-flex;
  border-radius: 999px;
  background: #241a16;
  padding: 0.18rem 0.55rem;
  color: #ffc8a5;
  font-size: 0.72rem;
  font-weight: 950;
}

.trouble-grid h3 {
  margin-top: 0.62rem;
  color: #201714;
  font-size: 1rem;
  font-weight: 950;
  line-height: 1.45;
}

.trouble-grid p {
  margin-top: 0.45rem;
  color: #6c5a52;
  line-height: 1.7;
}

.trouble-grid ol {
  margin-top: 0.7rem;
  font-size: 0.9rem;
}

.dark .tutorial-page,
.dark .hero-copy h1,
.dark .prep-card h2,
.dark .trouble-heading h2,
.dark .tutorial-step h2,
.dark .step-copy h3,
.dark .trouble-grid h3 {
  color: #fff7ed;
}

.dark .tutorial-hero {
  border-color: #312720;
  background:
    radial-gradient(circle at 86% 8%, rgba(232, 65, 46, 0.2), transparent 18rem),
    linear-gradient(135deg, #171311 0%, #0f0c0b 100%);
}

.dark .hero-copy p,
.dark .prep-card ol,
.dark .step-copy ol,
.dark .trouble-heading p,
.dark .trouble-grid p,
.dark .trouble-grid ol,
.dark .step-heading p {
  color: #cdbdb5;
}

.dark .prep-card,
.dark .troubleshooting,
.dark .tutorial-step,
.dark .step-index a,
.dark .tutorial-shot,
.dark .trouble-grid article {
  border-color: #312720;
  background: #171311;
}

.dark .step-index a {
  color: #f7ede4;
}

.dark .tutorial-shot figcaption {
  border-color: #312720;
  color: #cdbdb5;
}

.dark .tip-box {
  border-color: rgba(232, 65, 46, 0.24);
  background: rgba(232, 65, 46, 0.1);
}

.dark .tip-box p {
  color: #d8b7aa;
}

@media (max-width: 980px) {
  .tutorial-hero,
  .prep-card,
  .step-body,
  .trouble-grid {
    grid-template-columns: 1fr;
  }

  .step-index {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .tutorial-page {
    padding-bottom: 2rem;
  }

  .tutorial-hero,
  .prep-card,
  .troubleshooting,
  .step-heading,
  .step-body {
    border-radius: 18px;
    padding: 1rem;
  }

  .step-index {
    grid-template-columns: 1fr;
  }

  .step-heading {
    flex-direction: column;
  }
}
</style>
