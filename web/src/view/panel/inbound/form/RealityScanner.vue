<template>
  <Teleport to="body">
    <div v-if="modelValue" class="scanner-mask" @click.self="close">
      <section class="scanner">
        <header>
          <h2>Reality SNI 扫描</h2>
          <button class="icon-btn" title="关闭" @click="close"><i class="icon">close</i></button>
        </header>
        <div class="toolbar">
          <Input v-model="targets" placeholder="域名、IP 或 CIDR，多个目标用逗号分隔" />
          <button :disabled="loading" @click="scan">
            <i class="icon">radar</i>
            {{ loading ? '扫描中' : '扫描' }}
          </button>
        </div>
        <div class="results">
          <table>
            <thead>
              <tr>
                <th>目标</th>
                <th>状态</th>
                <th>TLS</th>
                <th>ALPN</th>
                <th>延迟</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="result in results" :key="result.target">
                <td>
                  <strong>{{ result.target }}</strong
                  ><small>{{ result.ip }}</small>
                </td>
                <td>
                  <span class="tag" :class="result.feasible ? 'green' : 'yellow'">{{
                    result.feasible ? '可用' : '检查'
                  }}</span
                  ><small v-if="!result.feasible" :title="result.reason">{{ result.reason }}</small>
                </td>
                <td>{{ result.tls_version || '-' }}</td>
                <td>{{ result.alpn || '-' }}</td>
                <td>{{ result.latency_ms ? `${result.latency_ms} ms` : '-' }}</td>
                <td>
                  <button
                    class="use"
                    :disabled="!result.feasible"
                    :title="result.feasible ? '使用此目标' : result.reason"
                    @click="pick(result)"
                  >
                    使用
                  </button>
                </td>
              </tr>
              <tr v-if="!loading && results.length === 0">
                <td colspan="6" class="empty">暂无扫描结果</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import Input from '@/component/ui/Input.vue'
import { scanRealityTargets } from '@/api/generate'

defineProps<{ modelValue: boolean }>()
const emit = defineEmits(['update:modelValue', 'select'])
const modal = inject<any>('modal')
const targets = ref('')
const results = ref<any[]>([])
const loading = ref(false)

function close() {
  emit('update:modelValue', false)
}

async function scan() {
  loading.value = true
  try {
    const response = await scanRealityTargets(targets.value)
    results.value = response.results || []
  } catch (error: any) {
    modal.value?.show('error', error?.error || '扫描失败')
  } finally {
    loading.value = false
  }
}

function pick(result: any) {
  emit('select', result)
  close()
}
</script>

<style scoped>
.scanner-mask {
  position: fixed;
  inset: 0;
  z-index: 1100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(0, 0, 0, 0.58);
  backdrop-filter: blur(5px);
}

.scanner {
  width: min(920px, 100%);
  max-height: min(720px, 90vh);
  display: flex;
  flex-direction: column;
  background: var(--color-bg-dark);
  border-radius: 8px;
  box-shadow: var(--box-shadow);
}

header,
.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-bg);
}

header {
  justify-content: space-between;
}

h2 {
  font-size: var(--font-size-md);
  color: var(--color-text-light);
}

.toolbar button,
.use {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 9px 16px;
  white-space: nowrap;
}

.results {
  overflow: auto;
  padding: 12px 20px 20px;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--font-size-sm);
}

th,
td {
  padding: 10px 8px;
  text-align: left;
  border-bottom: 1px solid var(--color-bg);
  color: var(--color-text);
}

th {
  color: var(--color-text-dark);
}

td small {
  display: block;
  max-width: 270px;
  margin-top: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-dark);
}

.empty {
  padding: 36px;
  text-align: center;
  color: var(--color-text-dark);
}

@media (max-width: 640px) {
  .scanner-mask {
    padding: 8px;
  }
  .scanner {
    max-height: 95vh;
  }
  .toolbar {
    align-items: stretch;
    flex-direction: column;
  }
  .results {
    padding-inline: 12px;
  }
}
</style>
