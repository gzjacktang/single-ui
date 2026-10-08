import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { runInNewContext } from 'node:vm'

const source = readFileSync(new URL('../src/view/panel/inbound/Inbound.vue', import.meta.url), 'utf8')

function extract(start, end) {
  const from = source.indexOf(start)
  assert.notEqual(from, -1, `missing ${start}`)
  const to = source.indexOf(end, from)
  assert.notEqual(to, -1, `missing ${end}`)
  return source.slice(from, to)
    .replace('catch (err: any)', 'catch (err)')
    .replace('(ib: Record<string, unknown>)', '(ib)')
}

const load = extract('async function load()', '\nasync function toggle(')
const save = extract('async function handleSave()', '\n</script>')
const inbound = { ID: 1, Enable: true, Name: '', Protocol: 'tunnel', Port: 10080, TunnelAddress: 'example.com', TunnelPort: 443, TunnelNetwork: 'tcp,udp' }

async function runScenario(saveInbound, getInbounds, invocation = 'handleSave()') {
  const messages = []
  const context = {
    defaultInbound: { value: { ...inbound, ID: 0 } },
    inbounds: { value: [] },
    showDrawer: { value: true },
    saving: { value: false },
    modal: { value: { show: (type, message) => messages.push({ type, message }) } },
    saveInbound,
    getInbounds,
    setTimeout: (callback) => callback(),
  }
  await runInNewContext(`${load}\n${save}\n${invocation}`, context)
  return { messages, context }
}

test('successful Tunnel save is not reclassified as failure when refresh disconnects', async () => {
  let persisted = false
  const { messages, context } = await runScenario(
    async () => { persisted = true; return inbound },
    async () => { throw new TypeError('Failed to fetch') },
  )
  assert.equal(persisted, true)
  assert.equal(messages.at(-1)?.type, 'success')
  assert.equal(context.showDrawer.value, false)
  assert.equal(context.inbounds.value.length, 1)
})

test('lost save response is reconciled from the persisted Tunnel record', async () => {
  const { messages, context } = await runScenario(
    async () => { throw new TypeError('Failed to fetch') },
    async () => [inbound],
  )
  assert.equal(messages.at(-1)?.type, 'success')
  assert.equal(context.showDrawer.value, false)
  assert.equal(context.inbounds.value.length, 1)
})

test('validation errors retain their message', async () => {
  const { messages } = await runScenario(
    async () => { throw { error: '端口已被占用' } },
    async () => { throw new Error('must not refresh after validation failure') },
  )
  assert.equal(messages.at(-1)?.type, 'error')
  assert.equal(messages.at(-1)?.message, '端口已被占用')
})

test('unconfirmed saves show an explicit message instead of an empty error', async () => {
  const { messages } = await runScenario(
    async () => { throw new TypeError('Failed to fetch') },
    async () => { throw new TypeError('Failed to fetch') },
  )
  assert.equal(messages.at(-1)?.type, 'error')
  assert.match(messages.at(-1)?.message || '', /无法确认/)
})

test('a pre-existing Tunnel with different settings is not mistaken for the saved record', async () => {
  const { messages } = await runScenario(
    async () => { throw new TypeError('Failed to fetch') },
    async () => [{ ...inbound, Name: '旧配置' }],
  )
  assert.match(messages.at(-1)?.message || '', /无法确认/)
})

test('a second save click does not send a duplicate request', async () => {
  let calls = 0
  const { messages } = await runScenario(
    async () => { calls++; return inbound },
    async () => [],
    'Promise.all([handleSave(), handleSave()])',
  )
  assert.equal(calls, 1)
  assert.equal(messages.length, 1)
})
