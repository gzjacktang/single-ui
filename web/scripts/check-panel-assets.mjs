import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'

const dist = new URL('../dist/', import.meta.url)
const html = readFileSync(new URL('index.html', dist), 'utf8')
const entry = html.match(/<script[^>]+src="(\.\/assets\/[^\"]+\.js)"/)

assert.ok(entry, '入口脚本必须使用相对资源路径，以支持任意面板路径')
assert.match(html, /href="\.\/assets\/[^\"]+\.css"/, '入口样式必须使用相对资源路径')

const script = readFileSync(join(fileURLToPath(dist), entry[1]), 'utf8')
assert.doesNotMatch(
  script,
  /\breturn\s*["'`]\/["'`]\s*\+\s*[\w$]+/,
  '延迟加载的菜单样式不能从站点根路径 /assets/ 请求',
)

console.log('面板入口和延迟加载资源路径检查通过')
