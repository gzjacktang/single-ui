import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { runInNewContext } from 'node:vm'
import ts from 'typescript'

const component = readFileSync(new URL('../src/component/widget/Task.vue', import.meta.url), 'utf8')
const script = component.match(/<script setup lang="ts">([\s\S]*?)<\/script>/)[1]
const executable = ts.transpile(script, { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 })
  .replace(/^import .*$/gm, '').replace(/^export \{\};?$/gm, '')

async function runTask(status) {
  const events = []
  const timers = []
  const cleanups = []
  const context = {
    defineProps: () => ({ taskId: 'cert-example' }),
    defineEmits: () => (event) => events.push(event),
    ref: value => ({ value }),
    watch: (value, callback) => callback(value()),
    nextTick: callback => callback?.(),
    onUnmounted: callback => cleanups.push(callback),
    task: () => ({}),
    getTaskStatus: async () => status,
    setInterval: callback => { timers.push(callback); return timers.length },
    clearInterval: () => {},
  }
  runInNewContext(executable, context)
  for (const timer of timers) await timer()
  await new Promise(resolve => setImmediate(resolve))
  for (const cleanup of cleanups) cleanup()
  return events
}

test('certificate task finishes even when the log WebSocket never delivers DONE', async () => {
  const events = await runTask({ logs: [], next: 1, done: true, error: '' })
  assert.deepEqual(events, ['done'])
})

test('completed certificate failure clears processing state without reporting success', async () => {
  const events = await runTask({ logs: [{ time: '', level: 'ERROR', message: '验证失败' }], next: 1, done: true, error: '验证失败' })
  assert.deepEqual(events, ['error'])
})
