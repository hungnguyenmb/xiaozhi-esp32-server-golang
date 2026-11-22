import messages from './server-vi.json'
import overrides from './vi-overrides.json'

const catalog = { ...messages, ...overrides }
const han = /[\u3400-\u9fff]/u
const escapeRegex = value => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
const patterns = Object.entries(catalog)
  .filter(([source]) => /%[sdv]/.test(source))
  .map(([source, target]) => ({
    expression: new RegExp('^' + source.split(/%[sdv]/).map(escapeRegex).join('([\\s\\S]*?)') + '$'),
    target
  }))

// Only call for system feedback. Never translate names, chat text, IDs or provider values.
export function translateServerMessage(value) {
  if (typeof value !== 'string' || !han.test(value)) return value
  if (catalog[value]) return catalog[value]
  for (const { expression, target } of patterns) {
    const match = value.match(expression)
    if (match) {
      let index = 0
      return target.replace(/%[sdv]/g, () => match[++index] ?? '')
    }
  }
  return value
}

export function localizeApiFeedback(data, url = '') {
  if (!data || typeof data !== 'object' || data instanceof Blob) return
  for (const field of ['error', 'message', 'msg']) {
    if (typeof data[field] === 'string') data[field] = translateServerMessage(data[field])
  }
  if (url.includes('/captcha/challenge') && typeof data.prompt === 'string') {
    data.prompt = translateServerMessage(data.prompt)
  }
  // Config tests return nested feedback, not user content.
  if (url.includes('/configs/test') && data.data) {
    for (const group of Object.values(data.data)) {
      for (const result of Object.values(group || {})) {
        if (result && typeof result.message === 'string') result.message = translateServerMessage(result.message)
      }
    }
  }
}
