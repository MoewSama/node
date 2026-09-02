<template>
    <main>
        <div class="page">
            <div class="body">
                <button class="add-btn" @click="openCreate">
                    <i class="icon">add</i>
                    添加出站
                </button>

                <List :headers="['ID', '备注', '协议', '落地', '启用', '操作']">
                    <tr v-for="ob in outbounds" :key="ob.ID">
                        <td class="muted">{{ ob.ID }}</td>
                        <td>{{ ob.Name }}</td>
                        <td><span class="tag primary">{{ ob.Protocol }}</span></td>
                        <td class="muted">{{ ob.Address }}:{{ ob.Port }}</td>
                        <td>
                            <Toggle :model-value="ob.Enable" @update:model-value="toggle(ob)" />
                        </td>
                        <td class="actions">
                            <button class="icon-btn" @click="openRoute(ob)" title="路由">
                                <i class="icon">alt_route</i>
                            </button>
                            <button class="icon-btn" @click="openEdit(ob)" title="编辑">
                                <i class="icon">edit</i>
                            </button>
                            <button class="icon-btn danger" @click="remove(ob.ID)" title="删除">
                                <i class="icon">delete</i>
                            </button>
                        </td>
                    </tr>
                    <tr v-if="outbounds.length === 0">
                        <td colspan="6" class="empty">暂无出站</td>
                    </tr>
                </List>
            </div>
        </div>
    </main>
    <Drawer v-model="showDrawer" :title="drawerTitle" @save="handleSave">
        <Form v-model="defaultOutbound" />
    </Drawer>
    <Drawer v-model="showRoute" title="路由设置" @save="routeRef?.save()">
        <Route v-if="showRoute" ref="routeRef" :tag="routeTag" />
    </Drawer>
</template>

<script setup lang="ts">
import Toggle from '@/component/ui/Toggle.vue'
import Drawer from '@/component/Drawer.vue'
import List from '@/component/ui/List.vue'
import Form from '@/view/panel/outbound/form/Form.vue'
import Route from '@/view/panel/endpoint/widget/Route.vue'
import { getOutbounds, saveOutbound, deleteOutbound, toggleOutbound } from '@/api/outbound'

const modal = inject<any>('modal')

const outbounds = ref<any[]>([])
const showDrawer = ref(false)
const drawerTitle = ref('添加出站')
const defaultOutbound = ref<any>(baseOutbound())

function baseOutbound() {
    return {
        Enable: true,
        Name: '',
        Protocol: 'vless',
        Address: '',
        Port: 443,
        UUID: '',
        Transport: 'raw',
        WsPath: '/',
        WsHost: '',
        TLSType: 'none',
        ServerName: '',
        Insecure: false,
        UTLS: '',
        ALPN: [],
    }
}

onMounted(load)

async function load() {
    outbounds.value = await getOutbounds()
}

async function toggle(ob: any) {
    await toggleOutbound(ob.ID)
    await load()
}

async function remove(id: number) {
    modal.value?.show('confirm', '确认删除该出站？关联的路由规则会一并清理。', async () => {
        await deleteOutbound(id)
        await load()
    })
}

function openCreate() {
    drawerTitle.value = '添加出站'
    defaultOutbound.value = baseOutbound()
    showDrawer.value = true
}

function openEdit(ob: any) {
    defaultOutbound.value = {
        ...baseOutbound(),
        ...ob,
        ALPN: ob.ALPN ? ob.ALPN.split(',') : [],
    }
    drawerTitle.value = '编辑出站'
    showDrawer.value = true
}

const showRoute = ref(false)
const routeTag = ref('')
const routeRef = ref<any>(null)

function openRoute(ob: any) {
    routeTag.value = ob.Name
    showRoute.value = true
}

async function handleSave() {
    const data = { ...defaultOutbound.value }
    if (Array.isArray(data.ALPN)) {
        data.ALPN = data.ALPN.join(',')
    }
    try {
        await saveOutbound(data)
        await load()
        showDrawer.value = false
        modal.value?.show('success', '保存成功')
    } catch (err: any) {
        modal.value?.show('error', err?.error)
    }
}
</script>
