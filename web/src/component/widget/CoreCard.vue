<template>
    <div class="card md">
        <div class="header">
            <div class="title">{{ core.Name }}</div>
            <div class="header-meta">
                <span class="version">v{{ updateInfo?.current_version || core.Version || '未知' }}</span>
                <div class="status">
                    <Status :status="status === 'running' ? 'yes' : 'no'" :animated="status === 'running'" />
                    <span>{{ status === 'running' ? '运行中' : '未启动' }}</span>
                </div>
            </div>
        </div>
        <p v-if="updateInfo?.has_update" class="update-hint">可更新至 sing-box {{ updateInfo.latest_version }}</p>
        <p v-else-if="checkError" class="update-hint error">{{ checkError }}</p>
        <div class="actions">
            <button class="action-btn" :disabled="updating" @click="status === 'running' ? handleStop() : handleStart()">
                <i class="icon">power_settings_new</i>
                {{ status === 'running' ? '停止' : '启动' }}
            </button>
            <button class="action-btn" :disabled="updating" @click="handleRestart">
                <i class="icon">refresh</i>
                重启
            </button>
            <button class="action-btn" :disabled="checking || updating" @click="checkForUpdate(true)">
                <i class="icon">search</i>
                {{ checking ? '检查中' : '检查更新' }}
            </button>
            <button class="action-btn" :disabled="!updateInfo?.has_update || updating" @click="handleUpdate">
                <i class="icon">system_update_alt</i>
                {{ updating ? '更新中' : updateInfo?.has_update ? '更新内核' : updateInfo ? '已是最新' : '暂不可用' }}
            </button>
        </div>
    </div>
</template>

<script setup lang="ts">
import Status from '@/component/ui/Status.vue'
import { getCore, getCoreStatus, stopCore, startCore, restartCore, checkCoreUpdate, updateCoreBinary, type CoreUpdateInfo } from '@/api/core'

const modal = inject<any>('modal')
const core = ref<any>({})
const status = ref<string>('stopped')
const updateInfo = ref<CoreUpdateInfo | null>(null)
const checking = ref(false)
const updating = ref(false)
const checkError = ref('')
let timer: ReturnType<typeof setInterval> | undefined

async function fetch() {
    const [c, s] = await Promise.all([getCore(), getCoreStatus()])
    core.value = c
    status.value = s.status
}

async function handleStart() {
    await startCore()
    await fetch()
}

async function handleStop() {
    await stopCore()
    status.value = 'stopped'
}

async function handleRestart() {
    await restartCore()
    await fetch()
}

async function checkForUpdate(showResult: boolean) {
    checking.value = true
    checkError.value = ''
    try {
        updateInfo.value = await checkCoreUpdate()
        if (showResult) {
            modal.value?.show('success', updateInfo.value.has_update
                ? `发现新版本 sing-box ${updateInfo.value.latest_version}`
                : '内核已是最新版本')
        }
    } catch (err: any) {
        updateInfo.value = null
        checkError.value = '内核更新检查失败，请稍后重试'
        if (showResult) modal.value?.show('error', err?.error || checkError.value)
    } finally {
        checking.value = false
    }
}

async function confirmInstalled(version: string) {
    for (let attempt = 0; attempt < 4; attempt++) {
        await new Promise(resolve => setTimeout(resolve, 1500))
        try {
            updateInfo.value = await checkCoreUpdate()
            await fetch()
            if (updateInfo.value.current_version === version) return true
        } catch { /* 核心重启时连接可能短暂中断 */ }
    }
    return false
}

function handleUpdate() {
    if (!updateInfo.value?.has_update || updating.value) return
    const target = updateInfo.value.latest_version
    modal.value?.show('confirm', `确认将 sing-box 内核更新至 ${target}？更新时节点会短暂中断。`, async () => {
        updating.value = true
        modal.value?.show('warn', '正在下载、校验并更新内核...')
        try {
            updateInfo.value = await updateCoreBinary()
            await fetch()
            modal.value?.show('success', `内核已更新至 ${updateInfo.value.current_version}`)
        } catch (err: any) {
            if (!err?.error && await confirmInstalled(target)) {
                modal.value?.show('success', `内核已更新至 ${target}`)
            } else {
                modal.value?.show('error', err?.error || '更新结果暂时无法确认，请稍后重新检查内核版本')
            }
        } finally {
            updating.value = false
        }
    })
}

onMounted(() => {
    fetch()
    checkForUpdate(false)
    timer = setInterval(fetch, 5000)
})
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>

<style scoped>
.header {
    justify-content: space-between;
    margin-bottom: 10px;

    .status {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: var(--font-size-sm);
        color: var(--color-text-dark);
    }
}

.header-meta {
    display: flex;
    align-items: center;
    gap: 14px;

    .version {
        color: var(--color-text-dark);
        font-size: var(--font-size-sm);
    }
}

.update-hint {
    margin: 0 0 12px;
    color: var(--color-primary);
    font-size: var(--font-size-sm);

    &.error { color: var(--color-red); }
}

.actions {
    display: flex;
    margin-top: auto;
    margin-left: calc(-1 * var(--card-padding, 20px));
    margin-right: calc(-1 * var(--card-padding, 20px));
    margin-bottom: calc(-1 * var(--card-padding, 20px));
    border-top: 1px solid var(--color-bg);
    overflow: hidden;

    .action-btn {
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 5px;
        padding: 10px;
        background: transparent;
        border-right: 1px solid var(--color-bg);
        color: var(--color-text);
        font-size: var(--font-size-sm);

        &:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }

        &:first-child { border-radius: 0 0 0 20px; }
        &:nth-child(2) { border-radius: 0; }
        &:last-child { 
            border-right: none; 
            border-radius: 0 0 20px 0;
        }

        &:not(:disabled):hover {
            background: var(--color-primary);
            color: var(--color-text-light);
        }
    }
}
</style>
