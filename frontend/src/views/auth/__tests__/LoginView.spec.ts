import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const getPublicSettings = vi.hoisted(() => vi.fn())
const login = vi.hoisted(() => vi.fn())
const login2FA = vi.hoisted(() => vi.fn())
const routerPush = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())
const showWarning = vi.hoisted(() => vi.fn())

const appStoreState = vi.hoisted(() => ({
    siteName: '烂番茄',
    siteLogo: '',
    publicSettingsLoaded: true,
    cachedPublicSettings: {
      site_subtitle: 'Subscription to API Conversion Platform',
    },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerPush,
    currentRoute: { value: { query: {} } },
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        const translations: Record<string, string> = {
          'auth.welcomeBack': '欢迎回来',
          'auth.signInToAccount': '登录您的账户以继续',
          'auth.emailLabel': '邮箱',
          'auth.emailPlaceholder': '请输入邮箱',
          'auth.passwordLabel': '密码',
          'auth.passwordPlaceholder': '请输入密码',
          'auth.signIn': '登录',
          'auth.signingIn': '登录中...',
          'auth.dontHaveAccount': '还没有账户？',
          'auth.signUp': '注册',
          'auth.loginSuccess': '登录成功',
          'auth.loginFailed': '登录失败',
          'auth.emailRequired': '请输入邮箱',
          'auth.invalidEmail': '请输入有效邮箱',
          'auth.passwordRequired': '请输入密码',
          'auth.passwordMinLength': '密码至少 6 位',
        }
        return translations[key] ?? key
      },
    }),
  }
})

vi.mock('@/api/auth', () => ({
  getPublicSettings,
  isTotp2FARequired: (response: unknown) =>
    typeof response === 'object' &&
    response !== null &&
    'requires_2fa' in response &&
    response.requires_2fa === true,
  isWeChatWebOAuthEnabled: () => false,
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    login,
    login2FA,
  }),
  useAppStore: () => ({
    ...appStoreState,
    fetchPublicSettings: vi.fn(async () => appStoreState.cachedPublicSettings),
    showError,
    showSuccess,
    showWarning,
  }),
}))

vi.mock('@/utils/oauthAffiliate', () => ({
  clearAllAffiliateReferralCodes: vi.fn(),
}))

vi.mock('@/utils/apiError', () => ({
  extractI18nErrorMessage: (_error: unknown, _t: unknown, _scope: string, fallback: string) => fallback,
}))

import LoginView from '../LoginView.vue'

const mountLoginView = () =>
  mount(LoginView, {
    global: {
      stubs: {
        Icon: { template: '<span class="icon-stub" />' },
        RouterLink: { props: ['to'], template: '<a><slot /></a>' },
        TurnstileWidget: true,
        LinuxDoOAuthSection: true,
        DingTalkOAuthSection: true,
        WechatOAuthSection: true,
        OidcOAuthSection: true,
        EmailOAuthButtons: true,
        LoginAgreementPrompt: true,
        TotpLoginModal: true,
      },
    },
  })

describe('LoginView product fit', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    appStoreState.siteName = '烂番茄'
    appStoreState.siteLogo = ''
    appStoreState.publicSettingsLoaded = true
    appStoreState.cachedPublicSettings = {
      site_subtitle: 'Subscription to API Conversion Platform',
    }
    getPublicSettings.mockResolvedValue({
      turnstile_enabled: false,
      turnstile_site_key: '',
      linuxdo_oauth_enabled: false,
      dingtalk_oauth_enabled: false,
      wechat_oauth_enabled: false,
      wechat_oauth_open_enabled: false,
      wechat_oauth_mp_enabled: false,
      backend_mode_enabled: false,
      oidc_oauth_enabled: false,
      oidc_oauth_provider_name: 'OIDC',
      github_oauth_enabled: false,
      google_oauth_enabled: false,
      password_reset_enabled: false,
      login_agreement_enabled: false,
      login_agreement_documents: [],
    })
  })

  it('renders the LANFANQIE writer-workbench login shell instead of the legacy API conversion copy', async () => {
    const wrapper = mountLoginView()
    await flushPromises()

    expect(wrapper.text()).toContain('LANFANQIE STUDIO')
    expect(wrapper.text()).toContain('烂番茄')
    expect(wrapper.text()).toContain('热榜、拆书、封面与创作生成一体化工作台')
    expect(wrapper.text()).toContain('登录创作台，继续你的热榜拆解和下一稿')
    expect(wrapper.text()).toContain('番茄热榜、封面生成、拆书诊断、爆款对标和创作生成')
    expect(wrapper.text()).toContain('热榜雷达')
    expect(wrapper.text()).toContain('封面工坊')
    expect(wrapper.text()).toContain('毒舌质检')
    expect(wrapper.text()).toContain('欢迎回来')
    expect(wrapper.text()).toContain('登录您的账户以继续')
    expect(wrapper.find('input[type="email"]').exists()).toBe(true)
    expect(wrapper.find('input[type="password"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('Subscription to API Conversion Platform')
  })
})
