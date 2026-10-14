<template>
    <div class="task" v-if="lines.length || statusError">
        <div class="output" ref="outputRef">
            <div v-if="statusError" class="message">{{ statusError }}</div>
            <div class="line" v-for="(line, i) in lines" :key="i">
                <div class="meta">
                    <span class="time">{{ line.time }}</span>
                    <span class="level" :class="line.level.toLowerCase()">{{ line.level }}</span>
                </div>
                <span class="message">{{ line.message }}</span>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { getTaskStatus } from '@/api/task'

const props = defineProps<{ taskId: string }>()
const emit = defineEmits<{ done: [], error: [] }>()

interface LogLine { time: string; level: string; message: string }

const lines = ref<LogLine[]>([])
const outputRef = ref<HTMLElement>()
const statusError = ref('')
let timer: ReturnType<typeof setInterval> | undefined
let generation = 0

function stop() {
    generation++
    if (timer) clearInterval(timer)
    timer = undefined
}

watch(() => props.taskId, (id) => {
    stop()
    if (!id) return
    lines.value = []
    statusError.value = ''
    const current = generation
    let cursor = 0
    let polling = false
    async function poll() {
        if (polling || current !== generation) return
        polling = true
        try {
            const status = await getTaskStatus(id, cursor)
            if (current !== generation) return
            lines.value = [...lines.value, ...status.logs].slice(-1000)
            cursor = status.next
            statusError.value = ''
            if (status.done) {
                stop()
                if (status.error) emit('error')
                else emit('done')
            }
            nextTick(() => outputRef.value?.scrollTo(0, outputRef.value.scrollHeight))
        } catch (err: any) {
            if (current !== generation) return
            statusError.value = err?.error || '任务进度暂时无法连接，正在重试...'
            if (err?.expired) {
                stop()
                emit('error')
                return
            }
        } finally {
            polling = false
        }
    }
    timer = setInterval(poll, 1500)
    poll()
}, { immediate: true })

onUnmounted(stop)
</script>

<style scoped>
.task {
    margin: 0 20px 20px;
    height: 500px;
    border-radius: 10px;
    overflow: hidden;
    background: var(--color-bg);

    .output {
        padding: 12px;
        max-height: 500px;
        overflow-y: auto;
        font-family: monospace;
        font-size: 12px;
        line-height: 1.8;

        .line {
            display: flex;
            gap: 10px;

            .meta {
                display: flex;
                flex-direction: row;
                flex-wrap: nowrap;
                gap: 10px;

                .time {
                    color: var(--color-text-dark);
                    white-space: nowrap;
                    flex-shrink: 0;
                }

                .level {
                    white-space: nowrap;
                    flex-shrink: 0;
                    font-weight: bold;

                    &.info  { color: var(--color-green); }
                    &.warn  { color: var(--color-yellow); }
                    &.error { color: var(--color-red); }
                }

            }

            .message {
                color: var(--color-text-light);
                word-break: break-all;
            }
        }
    }
}


@media (max-width: 768px) {
    .task {
        .output {
            .line {
                flex-direction: column;
                gap: 0;
            }
        }
    }
}
</style>
