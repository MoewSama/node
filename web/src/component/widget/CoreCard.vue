<template>
    <div class="card md">
        <div class="header">
            <div class="title">{{ core.Name }}</div>
            <div class="status">
                <Status :status="status === 'running' ? 'yes' : 'no'" :animated="status === 'running'" />
                <span>{{ status === 'running' ? '运行中' : '未启动' }}</span>
            </div>
        </div>
        <div class="actions">
            <button class="action-btn" @click="status === 'running' ? handleStop() : handleStart()">
                <i class="icon">power_settings_new</i>
                {{ status === 'running' ? '停止' : '启动' }}
            </button>
            <button class="action-btn" @click="handleRestart">
                <i class="icon">refresh</i>
                重启
            </button>
            <button class="action-btn version" :class="{ 'has-update': hasUpdate }" @click="openUpdate">
                <i class="icon">{{ hasUpdate ? 'new_releases' : 'commit' }}</i>
                {{ hasUpdate ? '有新核心' : core.Version }}
            </button>
        </div>
    </div>
    <Drawer v-model="showUpdate" :title="hasUpdate ? '发现新核心' : '强制更新核心'" saveText="立即更新" @save="handleUpdate">
        <Update :current-version="currentVersion" :latest-version="latestVersion" :changelog="hasUpdate ? 'sing-box 核心更新' : '当前已是最新版本，强制更新将重新下载并覆盖安装（相同版本号）'" />
    </Drawer>
</template>

<script setup lang="ts">
import Status from '@/component/ui/Status.vue'
import Drawer from '@/component/Drawer.vue'
import Update from '@/component/widget/Update.vue'
import { getCore, getCoreStatus, stopCore, startCore, restartCore, checkCoreUpdate, updateCoreBin } from '@/api/core'

const modal = inject<any>('modal')
const core = ref<any>({})
const status = ref<string>('stopped')
const hasUpdate = ref(false)
const currentVersion = ref('')
const latestVersion = ref('')
const showUpdate = ref(false)

async function fetch() {
    const [c, s] = await Promise.all([getCore(), getCoreStatus()])
    core.value = c
    status.value = s.status
}

async function fetchUpdate() {
    const res = await checkCoreUpdate()
    hasUpdate.value = res.has_update
    currentVersion.value = res.current_version
    latestVersion.value = res.latest_version
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

function openUpdate() {
    showUpdate.value = true
}

async function handleUpdate() {
    const force = !hasUpdate.value
    const tip = force
        ? '核心已是最新版本，强制更新将重新下载覆盖（用于重装）\n确认更新？'
        : '核心更新完成后将自动重启\n确认更新？'
    modal.value?.show('confirm', tip, async () => {
        modal.value?.update('warn', '更新中...')
        try {
            await updateCoreBin(force)
            modal.value?.update('success', '更新成功，正在重启核心...')
            await handleRestart()
            await fetchUpdate()
            showUpdate.value = false
        } catch (err: any) {
            modal.value?.update('error', err?.error ?? '更新失败')
        }
    })
}

onMounted(() => {
    fetch()
    fetchUpdate()
    const timer = setInterval(fetch, 5000)
    onUnmounted(() => clearInterval(timer))
})
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

        &:first-child { border-radius: 0 0 0 20px; }
        &:nth-child(2) { border-radius: 0; }
        &:last-child { 
            border-right: none; 
            border-radius: 0 0 20px 0;
        }

        &:hover {
            background: var(--color-primary);
            color: var(--color-text-light);
        }
    }
}
</style>