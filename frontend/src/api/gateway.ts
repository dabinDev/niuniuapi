/**
 * 网关直连客户端：用「用户自己的 API 密钥」直接调底层 OpenAI 兼容网关。
 *
 * 用途：在密钥管理页根据某把密钥获取其支持的模型、并对模型做生图/文案测试。
 * 鉴权用的是该密钥本身（Bearer），与面板会话 token 无关，因此用独立 axios 实例，
 * 不走 apiClient 的 {code,data} 拦截与会话注入。
 */

import axios from 'axios'

const gw = axios.create()

function authHeaders(apiKey: string) {
  return { headers: { Authorization: `Bearer ${apiKey}` } }
}

/** 通过密钥获取其支持的全部模型 id 列表（网关 GET /v1/models）。 */
export async function gwListModels(apiKey: string): Promise<string[]> {
  const { data } = await gw.get<{ data?: { id: string }[] }>('/v1/models', authHeaders(apiKey))
  return (data.data || []).map((m) => m.id)
}

/** 生图测试：用最小请求验证该密钥+模型可生图（失败抛错）。 */
export async function gwTestImage(apiKey: string, model: string): Promise<void> {
  await gw.post(
    '/v1/images/generations',
    { model, prompt: 'test', n: 1, size: '1024x1024' },
    authHeaders(apiKey),
  )
}

/** 文案测试：用最小聊天请求验证该密钥+模型可用（失败抛错）。 */
export async function gwTestChat(apiKey: string, model: string): Promise<void> {
  await gw.post(
    '/v1/chat/completions',
    { model, messages: [{ role: 'user', content: 'ping' }], max_tokens: 1 },
    authHeaders(apiKey),
  )
}

export const gatewayAPI = { gwListModels, gwTestImage, gwTestChat }
export default gatewayAPI
