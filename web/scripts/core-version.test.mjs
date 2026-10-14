import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import test from 'node:test'
import { runInNewContext } from 'node:vm'
import ts from 'typescript'

const component = readFileSync(new URL('../src/component/widget/CoreCard.vue', import.meta.url), 'utf8')
const script = component.match(/<script setup lang="ts">([\s\S]*?)<\/script>/)[1]
const executable = ts.transpile(script, { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 })
  .replace(/^import .*$/gm, '').replace(/^export \{\};?$/gm, '')

async function install(target, preview) {
  const pending = []
  const requested = []
  const info = { current_version: '1.14.3', latest_version: '1.14.3', has_update: false, versions: [
    { version: '1.14.3', prerelease: false }, { version: '1.13.13', prerelease: false }, { version: '1.15.0-alpha.12', prerelease: true },
  ] }
  const context = {
    target, preview, info,
    ref: value => ({ value }),
    computed: getter => ({ get value() { return getter() } }),
    watch: () => {}, onMounted: () => {}, onUnmounted: () => {},
    inject: () => ({ value: { show: (type, message, action) => { if (type === 'confirm') pending.push(action()) } } }),
    updateCoreBinary: async version => { requested.push(version); return { ...info, current_version: version } },
    getCore: async () => ({}), getCoreStatus: async () => ({ status: 'stopped' }),
  }
  runInNewContext(`${executable}\nupdateInfo.value = info; selectedVersion.value = target; includePreview.value = preview; handleUpdate()`, context)
  await Promise.all(pending)
  return requested
}

test('version picker can roll back even when no newer stable release exists', async () => {
  assert.deepEqual(await install('1.13.13', false), ['1.13.13'])
})
test('preview must be explicitly enabled and selected', async () => {
  assert.deepEqual(await install('1.15.0-alpha.12', false), [])
  assert.deepEqual(await install('1.15.0-alpha.12', true), ['1.15.0-alpha.12'])
})
test('fresh installs choose latest stable even after preview or rollback releases are published', () => {
  const installer = readFileSync(new URL('../../install.sh', import.meta.url), 'utf8')
  const assignment = installer.split('\n').find(line => line.startsWith('CORE_VERSION='))
  const fixture = [
    { tag_name: 'sing-box-1.15.0-alpha.12', prerelease: true },
    { tag_name: 'sing-box-1.13.13' }, { tag_name: 'sing-box-1.14.3' },
  ]
  const result = spawnSync('bash', ['-c', `curl() { printf '%s' "$CORE_RELEASE_FIXTURE"; }\n${assignment}\nprintf '%s' "$CORE_VERSION"`], {
    encoding: 'utf8', env: { ...process.env, CORE_RELEASE_FIXTURE: JSON.stringify(fixture, null, 2) },
  })
  assert.equal(result.status, 0, result.stderr)
  assert.equal(result.stdout, '1.14.3')
})
