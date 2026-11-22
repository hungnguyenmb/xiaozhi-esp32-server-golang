import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { createHash } from 'node:crypto'
import { fileURLToPath } from 'node:url'
import { parse as parseSfc } from '@vue/compiler-sfc'
import { parse as parseJs } from '@babel/parser'

const catalog = JSON.parse(fs.readFileSync(new URL('../src/locales/vi.json', import.meta.url)))
const serverCatalog = JSON.parse(fs.readFileSync(new URL('../src/locales/server-vi.json', import.meta.url)))
const overrides = JSON.parse(fs.readFileSync(new URL('../src/locales/vi-overrides.json', import.meta.url)))
const moduleSource = fs.readFileSync(new URL('../src/locales/server.js', import.meta.url),'utf8')
  .replace("import messages from './server-vi.json'", `const messages = ${JSON.stringify(serverCatalog)}`)
  .replace("import overrides from './vi-overrides.json'", `const overrides = ${JSON.stringify(overrides)}`)
const { translateServerMessage, localizeApiFeedback } = await import('data:text/javascript;base64,' + Buffer.from(moduleSource).toString('base64'))

test('Vietnamese feedback preserves user content and technical values', () => {
  const data = { error:'设备不存在', data:{ name:'小智', voice:'sage', device_code:'277167', content:'你好' } }
  const original = structuredClone(data.data)
  localizeApiFeedback(data, '/user/devices')
  assert.notEqual(data.error, '设备不存在')
  assert.doesNotMatch(data.error, /[\u3400-\u9fff]/u)
  assert.deepEqual(data.data, original)
  assert.equal(translateServerMessage('unknown voice sage'), 'unknown voice sage')
  assert.equal(translateServerMessage('Nội dung của người dùng'), 'Nội dung của người dùng')
})

test('Vietnamese API examples remain valid JSON with the same keys', () => {
  for (const [source,target] of Object.entries(catalog)) {
    let original
    try { original = JSON.parse(source) } catch { continue }
    const translated = JSON.parse(target)
    function keys(value) {
      if (Array.isArray(value)) return value.map(keys)
      if (value && typeof value === 'object') return Object.entries(value).map(([key,v])=>[key,keys(v)])
      return typeof value
    }
    assert.deepEqual(keys(translated), keys(original), source)
  }
})

test('nested provider test messages are translated without changing IDs or metrics', () => {
  const data = { data: { tts: { vieneu_tts: { ok:true, message:'通过', first_packet_ms:1200 } } } }
  localizeApiFeedback(data, '/admin/configs/test')
  assert.deepEqual(data.data.tts.vieneu_tts, { ok:true, message:'Đạt', first_packet_ms:1200 })
})

test('localization preserves provider/model/voice IDs and option values in source', () => {
  const frontend = fileURLToPath(new URL('..',import.meta.url))
  const repo = path.resolve(frontend,'../..')
  const baseline = JSON.parse(fs.readFileSync(new URL('./fixtures/localization-wire-values.json',import.meta.url)))
  const protectedKeys = new Set(['value','voice','voice_id','voiceId','model','model_name','provider','type','id','key','name','language','transcript_lang'])
  function values(file,source) {
    let blocks = [source]
    if (file.endsWith('.vue')) {
      const { descriptor } = parseSfc(source)
      blocks = [descriptor.script?.content,descriptor.scriptSetup?.content].filter(Boolean)
    }
    const result=[]
    function walk(node,protectedValue=false) {
      if (!node || typeof node !== 'object') return
      if (Array.isArray(node)) {node.forEach(x=>walk(x,protectedValue));return}
      if (node.type==='ObjectProperty') {
        walk(node.value,protectedValue||protectedKeys.has(node.key.name??node.key.value));return
      }
      if (protectedValue && node.type==='StringLiteral') result.push(node.value)
      for (const [k,v] of Object.entries(node)) if (!['loc','extra','comments','leadingComments','trailingComments'].includes(k) && v && typeof v==='object') walk(v,protectedValue)
    }
    for (const block of blocks) walk(parseJs(block,{sourceType:'module'}))
    return result
  }
  for (const [file, expected] of Object.entries(baseline)) {
    const current=fs.readFileSync(path.join(repo,file),'utf8')
    const actual=createHash('sha256').update(JSON.stringify(values(file,current))).digest('hex')
    assert.equal(actual,expected,file)
  }
})
