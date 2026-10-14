import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { runInNewContext } from 'node:vm'
import ts from 'typescript'

const source = readFileSync(new URL('../src/util/panelAddress.ts', import.meta.url), 'utf8')
const executable = ts.transpile(source, { target: ts.ScriptTarget.ES2022 }).replace('export ', '')
function address(origin, config, login = false) {
  return runInNewContext(`${executable}\npanelAddress(config, login)`, { URL, window: { location: new URL(origin) }, config, login })
}

test('restart after changing IP access port uses new port and nested panel path', () => {
  assert.equal(address('http://192.0.2.1:2053', { Port: 30000, Path: '/private/admin/panel' }, true), 'http://192.0.2.1:30000/private/admin/panel/login')
})
test('domain and IPv6 addresses preserve host and TLS choice', () => {
  assert.equal(address('http://192.0.2.1:2053', { Domain: 'panel.example.com', Port: 443, Path: '/admin' }, true), 'https://panel.example.com/admin/login')
  assert.equal(address('http://[2001:db8::1]:2053', { Port: 30000, Path: '/admin' }), 'http://[2001:db8::1]:30000/admin')
})
