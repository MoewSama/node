<template>
    <div class="outbound-form">
        <!-- 基础配置 -->
        <Section title="基础配置">
            <div class="form-row">
                <span class="form-label">启用</span>
                <Toggle v-model="form.Enable" />
            </div>
            <div class="form-row">
                <span class="form-label">备注</span>
                <Input v-model="form.Name" placeholder="同时作为出站 tag，需唯一" />
            </div>
            <div class="form-row">
                <span class="form-label">协议</span>
                <Select :model-value="'vless'" :options="[{ label: 'VLESS', value: 'vless' }]" disabled />
            </div>
            <div class="form-row">
                <span class="form-label">落地地址</span>
                <Input v-model="form.Address" placeholder="域名或 IP，支持 IPv6" />
            </div>
            <div class="form-row quarter">
                <span class="form-label">端口</span>
                <Input v-model="form.Port" type="number" :min="1" :max="65535" placeholder="1 - 65535" />
            </div>
            <div class="form-row">
                <span class="form-label">UUID</span>
                <Input v-model="form.UUID" placeholder="落地节点的用户 UUID" />
            </div>
        </Section>

        <!-- 传输 -->
        <Section title="传输">
            <div class="form-row half">
                <span class="form-label">传输</span>
                <Select v-model="form.Transport" :options="[
                    { label: 'RAW', value: 'raw' },
                    { label: 'WebSocket', value: 'websocket' },
                ]" />
            </div>
            <template v-if="form.Transport === 'websocket'">
                <div class="form-row">
                    <span class="form-label">路径</span>
                    <Input v-model="form.WsPath" placeholder="/path" />
                </div>
                <div class="form-row">
                    <span class="form-label">Host</span>
                    <Input v-model="form.WsHost" placeholder="example.com" />
                </div>
            </template>
        </Section>

        <!-- 安全 -->
        <Section title="安全">
            <div class="form-row half">
                <span class="form-label">安全</span>
                <RadioGroup v-model="form.TLSType" :options="[
                    { label: '无', value: 'none' },
                    { label: 'TLS', value: 'tls' },
                ]" />
            </div>
            <template v-if="form.TLSType === 'tls'">
                <div class="form-row">
                    <span class="form-label">SNI</span>
                    <Input v-model="form.ServerName" placeholder="example.com" />
                </div>
                <div class="form-row half">
                    <span class="form-label">uTLS</span>
                    <Select v-model="form.UTLS" :options="utlsOptions" />
                </div>
                <div class="form-row">
                    <span class="form-label">ALPN</span>
                    <MultiSelect v-model="form.ALPN" :options="[
                        { label: 'h2', value: 'h2' },
                        { label: 'http/1.1', value: 'http/1.1' },
                    ]" />
                </div>
                <div class="form-row">
                    <span class="form-label">跳过验证</span>
                    <Toggle v-model="form.Insecure" />
                </div>
            </template>
            <div class="form-tip">
                出站用于中转链：如「Cloudflared 隧道入站 → 本出站 → 落地机」。在端点页的路由绑定里，将入站与该出站关联。
            </div>
        </Section>
    </div>
</template>

<script setup lang="ts">
import Section from '@/component/ui/Section.vue'
import Toggle from '@/component/ui/Toggle.vue'
import Input from '@/component/ui/Input.vue'
import Select from '@/component/ui/Select.vue'
import RadioGroup from '@/component/ui/Radio.vue'
import MultiSelect from '@/component/ui/MultiSelect.vue'

const form = defineModel<any>({ default: () => ({
    Enable: true,
    Name: '',
    Protocol: 'vless',
    Address: '',
    Port: 443,
    UUID: '',
}) })

const utlsOptions = [
    { label: '不启用', value: '' },
    { label: 'Chrome', value: 'chrome' },
    { label: 'Firefox', value: 'firefox' },
    { label: 'Safari', value: 'safari' },
    { label: 'iOS', value: 'ios' },
    { label: 'Android', value: 'android' },
    { label: 'Edge', value: 'edge' },
    { label: 'Random', value: 'random' },
    { label: 'Randomized', value: 'randomized' },
]
</script>

<style scoped>
.outbound-form {
    display: flex;
    flex-direction: column;
}

:deep(.section) {
    border-radius: 0;

    .header {
        background-color: var(--color-bg);
    }
    .body {
        .content {
            display: flex;
            flex-direction: column;
            gap: 10px;
            padding: 20px;
        }
    }
}

.form-row {
    display: flex;
    align-items: center;
    gap: 16px;

    &.half :deep(.input-wrap),
    &.half :deep(.select) {
        max-width: 50%;
    }
}

.form-label {
    width: 100px;
    flex-shrink: 0;
    font-size: var(--font-size-sm);
    color: var(--color-text-dark);
    text-align: right;
}

.form-tip {
    padding: 10px 20px;
    font-size: var(--font-size-sm);
    color: var(--color-text-dark);
    opacity: 0.7;
    line-height: 1.6;
}
</style>
