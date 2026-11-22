// Inventory/apply translations to human-facing source nodes, never to API keys or regexes.
import fs from 'node:fs'
import path from 'node:path'
import { parse as parseSfc } from '@vue/compiler-sfc'
import { baseParse } from '@vue/compiler-dom'
import { parse as parseJs, parseExpression } from '@babel/parser'

const root = process.cwd()
const catalogArg = process.argv.indexOf('--catalog')
const suppliedCatalog = catalogArg >= 0 ? JSON.parse(fs.readFileSync(process.argv[catalogArg+1],'utf8')) : null
const hasHan = value => /[\u3400-\u9fff]/u.test(value) || (suppliedCatalog && Object.hasOwn(suppliedCatalog,value))
const protectedKeys = new Set(['value', 'voice', 'voice_id', 'voiceId', 'model', 'model_name', 'provider', 'type', 'id', 'key', 'name', 'language', 'transcript_lang', 'regex', 'pattern'])
const entries = []
function files(dir) {
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap(e => e.isDirectory() ? files(path.join(dir,e.name)) : [path.join(dir,e.name)])
}
function add(file, start, end, text, kind, quote = "'") {
  if (!hasHan(text) || !text.trim()) return
  entries.push({ file: path.relative(root,file), start, end, text, kind, quote })
}
function jsNodes(file, code, offset, expression=false) {
  let ast
  try { ast = expression ? parseExpression(code) : parseJs(code, { sourceType:'module', plugins:['typescript'] }) }
  catch (error) { throw new Error(`${file}: ${error.message}`) }
  function visit(node, parent, property, blocked=false) {
    if (!node || typeof node !== 'object') return
    if (Array.isArray(node)) { node.forEach(n => visit(n,parent,property,blocked)); return }
    const key = parent?.key?.name ?? parent?.key?.value
    blocked ||= parent?.type === 'ObjectProperty' && (property === 'key' || protectedKeys.has(key))
    blocked ||= parent?.type === 'BinaryExpression' && ['===','!==','==','!='].includes(parent.operator)
    blocked ||= node.type === 'CallExpression' && node.callee?.object?.name === 'console'
    if (!blocked && node.type === 'StringLiteral') add(file,offset+node.start,offset+node.end,node.value,'js',code[node.start])
    if (!blocked && node.type === 'TemplateElement') add(file,offset+node.start,offset+node.end,node.value.raw,'template')
    for (const [k,v] of Object.entries(node)) {
      if (['loc','extra','comments','leadingComments','trailingComments','innerComments','tokens'].includes(k)) continue
      if (v && typeof v === 'object') visit(v,node,k,blocked)
    }
  }
  visit(ast,null,'')
}
for (const file of files(path.join(root,'src')).filter(f=>/\.(vue|js)$/.test(f))) {
  const source = fs.readFileSync(file,'utf8')
  if (file.endsWith('.js')) { jsNodes(file,source,0); continue }
  const { descriptor, errors } = parseSfc(source)
  if (errors.length) throw errors[0]
  for (const block of [descriptor.script,descriptor.scriptSetup].filter(Boolean)) jsNodes(file,block.content,block.loc.start.offset)
  if (!descriptor.template) continue
  const base = descriptor.template.loc.start.offset
  const ast = baseParse(descriptor.template.content)
  function visit(node) {
    if (node.type === 2) {
      const raw = node.loc.source
      const text = raw.trim()
      const leading = raw.indexOf(text)
      add(file,base+node.loc.start.offset+leading,base+node.loc.start.offset+leading+text.length,text,'text')
    }
    if (node.type === 5) jsNodes(file,node.content.content,base+node.content.loc.start.offset,true)
    for (const prop of node.props || []) {
      if (prop.type === 6 && prop.value && !protectedKeys.has(prop.name)) {
        add(file,base+prop.value.loc.start.offset,base+prop.value.loc.end.offset,prop.value.content,'attr',prop.value.loc.source[0])
      } else if (prop.type === 7 && prop.exp && hasHan(prop.exp.content) && !(prop.name === 'bind' && protectedKeys.has(prop.arg?.content))) {
        // Statements in event handlers are parsed as a program; other directives as expressions.
        jsNodes(file,prop.exp.content,base+prop.exp.loc.start.offset,prop.name !== 'on')
      }
    }
    for (const child of node.children || []) visit(child)
  }
  visit(ast)
}
const localeDir = path.join(root,'src/locales')
fs.mkdirSync(localeDir,{recursive:true})
if (!process.argv.includes('--apply')) {
  fs.writeFileSync('/tmp/xiaozhi-vi-nodes.json',JSON.stringify(entries,null,2))
  fs.writeFileSync('/tmp/xiaozhi-vi-strings.json',JSON.stringify([...new Set(entries.map(e=>e.text))],null,2))
  console.log(JSON.stringify({nodes:entries.length,strings:new Set(entries.map(e=>e.text)).size,files:new Set(entries.map(e=>e.file)).size}))
} else {
  const dictionary = suppliedCatalog || JSON.parse(fs.readFileSync(path.join(localeDir,'vi.json'),'utf8'))
  const escapeHtml = s => s.replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;')
  for (const file of new Set(entries.map(e=>e.file))) {
    let source = fs.readFileSync(path.join(root,file),'utf8')
    const seen = new Set()
    for (const e of entries.filter(e=>e.file===file).sort((a,b)=>b.start-a.start)) {
      const position = `${e.start}:${e.end}`
      if (seen.has(position)) continue
      seen.add(position)
      const translated = dictionary[e.text]
      if (!translated || translated===e.text) continue
      let output = translated
      if (e.kind === 'js') output = e.quote + translated.replaceAll('\\','\\\\').replaceAll(e.quote,'\\'+e.quote).replaceAll('\n','\\n').replaceAll('\r','\\r') + e.quote
      if (e.kind === 'template') output = translated.replaceAll('`','\\`').replaceAll('${','\\${')
      if (e.kind === 'text') output = escapeHtml(translated)
      if (e.kind === 'attr') output = e.quote+escapeHtml(translated).replaceAll(e.quote,e.quote==='"'?'&quot;':'&#39;')+e.quote
      source=source.slice(0,e.start)+output+source.slice(e.end)
    }
    fs.writeFileSync(path.join(root,file),source)
  }
  console.log('Applied Vietnamese catalog to source nodes')
}
