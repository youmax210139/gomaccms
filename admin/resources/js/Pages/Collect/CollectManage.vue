<template>
  <div>
    <p v-if="page.props.errors?.form" style="color: #f56c6c; padding: 4px 0;">{{ page.props.errors.form }}</p>
    <TableToolbar>
      <el-button :icon="Plus" @click="router.get('/manage/collect/add')">添加</el-button>
      <el-button :icon="Delete" :disabled="selected.length === 0" @click="delSelected">删除</el-button>
      <el-button :icon="Delete" @click="clearBinds">清空绑定</el-button>
    </TableToolbar>
    <el-table stripe :data="siteList" style="width: 100%" size="default" row-key="id" table-layout="auto" @selection-change="(rows: any[]) => selected = rows">
      <el-table-column type="selection" width="48"/>
      <el-table-column type="index" label="编号" width="70"/>
      <el-table-column label="接口类型" width="100" align="center">
        <template #default="scope">
          <el-tag :type="scope.row.resultModel == 1 ? 'warning' : 'primary'" disable-transitions>{{ scope.row.resultModel == 1 ? 'XML' : 'JSON' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="资源类型" width="100" align="center">
        <template #default="scope">
          <el-tag type="success" disable-transitions>{{ collectTypeLabel(scope.row.collectType) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="资源站">
        <template #default="scope">
          <el-link underline="never" title="点击复制接口地址" @click="copyLink(scope.row.uri + (scope.row.params ?? ''))">【{{ scope.row.name }}】{{ scope.row.uri }}{{ scope.row.params }}</el-link>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90" align="center">
        <template #default="scope">
          <el-switch :model-value="scope.row.state" inline-prompt active-text="启用" inactive-text="停用" @change="(v: string | number | boolean) => changeState(scope.row, Boolean(v))" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="360">
        <template #default="scope">
          <el-button size="small" @click="router.get('/manage/collect/browse', { id: scope.row.id })">绑定</el-button>
          <el-button v-if="!can('collect.run')" size="small" @click="onCollectCommand(scope.row, 'logs')">采集记录</el-button>
          <el-dropdown v-else trigger="click" :disabled="!scope.row.state || isCollecting(scope.row.id)" class="op-dropdown" @command="(d: any) => onCollectCommand(scope.row, d)">
            <el-button size="small" :disabled="!scope.row.state" :loading="isCollecting(scope.row.id)"
                       @click="isCollecting(scope.row.id) && openCollectTask(scope.row.id)">
              {{ isCollecting(scope.row.id) ? '采集中' : '采集' }}<el-icon v-if="!isCollecting(scope.row.id)" class="el-icon--right"><ArrowDown/></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-for="d in collectDuration" :key="d.time" :command="d">{{ d.label }}</el-dropdown-item>
                <el-dropdown-item v-if="canResume(collectTasks.tasks[scope.row.id])" command="resume" divided>从中断处继续</el-dropdown-item>
                <el-dropdown-item command="logs" :divided="!canResume(collectTasks.tasks[scope.row.id])">采集记录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-button size="small" @click="router.get('/manage/collect/edit', { id: scope.row.id })">编辑</el-button>
          <el-button size="small" @click="delSourceSite(scope.row)">删除</el-button>
          <el-button v-if="can('collect.clear')" size="small" type="danger" plain @click="openClear(scope.row)">清空</el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog v-model="logsV" width="1000px" :title="`采集记录 · ${logsSource?.name ?? ''}`">
      <el-table stripe v-loading="logsLoading" :data="logs" size="small" empty-text="暂无采集记录 (每个采集接口保留最近 50 次)">
        <el-table-column label="开始时间" width="160">
          <template #default="{ row }">{{ formatDateTime(row.startedAt) }}</template>
        </el-table-column>
        <el-table-column prop="mode" label="方式" width="110"/>
        <el-table-column prop="trigger" label="触发" width="80" align="center"/>
        <el-table-column label="结果" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="collectStates[row.state]?.type ?? 'info'" disable-transitions>{{ collectStates[row.state]?.label ?? row.state }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="新增 / 更新 / 跳过" width="130" align="center">
          <template #default="{ row }">{{ row.added }} / {{ row.updated }} / {{ row.skipped }}</template>
        </el-table-column>
        <el-table-column prop="failedPages" label="失败页" width="70" align="center"/>
        <el-table-column label="耗时" width="90" align="center">
          <template #default="{ row }">{{ duration(row.startedAt, row.finishedAt) }}</template>
        </el-table-column>
        <el-table-column prop="message" label="说明" show-overflow-tooltip/>
        <el-table-column v-if="can('collect.run')" label="操作" width="110" align="center">
          <template #default="{ row }">
            <el-button v-if="row.failedList?.length && !row.retried" size="small" :disabled="isCollecting(row.sourceId)"
                       @click="retryLog(row)">重试失败页</el-button>
            <span v-else-if="row.retried" class="muted">已重试</span>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="logsV = false">关闭</el-button>
      </template>
    </AdminDialog>

    <AdminDialog v-model="clearV" width="520px" :title="`清空「${clearSource?.name ?? ''}」的视频`" confirm-text="确认清空" :loading="clearing" @confirm="clearVideos">
      <div v-loading="!clearStat" class="clear-stat">
        <template v-if="clearStat">
          <p v-if="clearStat.collecting" class="clear-warn">该采集接口正在采集中, 请等待采集完成后再清空。</p>
          <p>将删除 <b>{{ clearStat.exclusive }}</b> 部只来自此采集接口的视频 (连同其分类与来源记录)。</p>
          <p><b>{{ clearStat.shared }}</b> 部与其他采集接口共用的视频会保留, 只移除此采集接口提供的播放线路。</p>
        </template>
      </div>
      <el-form @submit.prevent>
        <el-form-item label="确认密码"><el-input v-model="password" type="password" placeholder="请输入当前账户密码" autocomplete="off" show-password /></el-form-item>
      </el-form>
    </AdminDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { router, usePage } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import { ArrowDown, Delete, Plus } from '@element-plus/icons-vue'
import { http } from '../../utils/http'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import TableToolbar from '../../Components/TableToolbar.vue'
import AdminDialog from '../../Components/AdminDialog.vue'
import { confirmAction } from '../../utils/confirm'
import { useCan } from '../../utils/menu'
import { formatDateTime } from '../../utils/format'
import {
    canResume, collectStates, collectTasks, isCollecting, openCollectTask, resumeCollect, retryFailedPages, startCollect,
} from '../../utils/collectTasks'

const can = useCan()

const props = defineProps<{ list: any[] }>()
const page = usePage<{ errors?: { form?: string } }>()

defineOptions({ layout: AdminLayout })

const collectTypeLabel = (t: number) => (['视频', '文章', '演员', '角色', '网站'][t] ?? '未知')

const siteList = computed(() => props.list ?? [])
const selected = ref<any[]>([])

const collectDuration = [
    { time: 24, label: '采集当天' },
    { time: 24 * 7, label: '采集本周' },
    { time: -1, label: '采集所有' },
]

const password = ref('')

// 点击资源站只复制接口地址; 非 https 部署时 navigator.clipboard 不可用, 退回 execCommand
const copyLink = async (text: string) => {
    try {
        await navigator.clipboard.writeText(text)
    } catch {
        const ta = document.createElement('textarea')
        ta.value = text
        ta.style.cssText = 'position:fixed;opacity:0'
        document.body.appendChild(ta)
        ta.select()
        const ok = document.execCommand('copy')
        ta.remove()
        if (!ok) return ElMessage.error('复制失败, 请手动复制')
    }
    ElMessage.success('已复制接口地址')
}

const delSourceSite = async (row: any) => {
    if (!await confirmAction(`确认删除采集接口「${row.name}」?`)) return
    router.get('/manage/collect/del', { id: row.id })
}

const delSelected = async () => {
    if (!await confirmAction(`确认删除选中的 ${selected.value.length} 个采集接口?`)) return
    router.post('/manage/collect/del', { ids: selected.value.map((r) => r.id) })
}

const changeState = (row: any, state: boolean) => {
    router.post('/manage/collect/change', { id: row.id, state }, { preserveScroll: true })
}

const clearBinds = async () => {
    const n = selected.value.length
    if (!await confirmAction(n ? `确认清空选中的 ${n} 个采集接口的分类绑定?` : '未选择采集接口, 确认清空全部采集接口的分类绑定?')) return
    router.post('/manage/collect/bind/clear', { ids: selected.value.map((r) => r.id) })
}

const onCollectCommand = (row: any, cmd: any) => {
    if (cmd === 'resume') resumeCollect(row.id)
    else if (cmd === 'logs') openLogs(row)
    else startTask(row, cmd)
}

// ---- 采集记录
const logsV = ref(false)
const logsLoading = ref(false)
const logsSource = ref<any>(null)
const logs = ref<any[]>([])
const openLogs = async (row: any) => {
    logsSource.value = row
    logs.value = []
    logsV.value = true
    logsLoading.value = true
    try {
        const resp = await http.get('/manage/collect/logs', { params: { id: row.id } })
        logs.value = resp.data.code === 0 ? (resp.data.data ?? []) : []
    } finally {
        logsLoading.value = false
    }
}
// 重采这次采集失败的页; 开始后关闭记录弹窗, 显示进度弹窗
const retryLog = async (row: any) => {
    const n = row.failedList.length
    if (!await confirmAction(`确认重新采集这次失败的 ${n} 页? 页码按采集站当前的数据重新请求。`)) return
    if (await retryFailedPages(row.sourceId, row.id)) logsV.value = false
}
const duration = (start: string, end: string) => {
    const s = Math.max(0, Math.round((new Date(end).getTime() - new Date(start).getTime()) / 1000))
    return s >= 60 ? `${Math.floor(s / 60)} 分 ${s % 60} 秒` : `${s} 秒`
}

// 开始采集后弹窗显示进度 (见 Components/CollectTasks.vue); 「采集所有」请求较多, 先确认
const startTask = async (row: any, d: { time: number; label: string }) => {
    if (d.time < 0 && !await confirmAction(`确认对「${row.name}」执行${d.label}? 会请求该接口全部页码, 耗时较长。`)) return
    startCollect(row.id, d.time)
}

// ---- 清空采集接口的视频
const clearV = ref(false)
const clearing = ref(false)
const clearSource = ref<any>(null)
const clearStat = ref<{ exclusive: number; shared: number; collecting: boolean } | null>(null)
const openClear = (row: any) => {
    clearSource.value = row
    clearStat.value = null
    password.value = ''
    clearV.value = true
    http.get('/manage/collect/clear/preview', { params: { id: row.id } }).then((resp: any) => {
        if (resp.data.code === 0) {
            clearStat.value = resp.data.data
        } else {
            ElMessage.error({ message: resp.data.msg })
        }
    })
}
const clearVideos = async () => {
    if (!password.value) {
        ElMessage.warning({ message: '请输入当前账户密码' })
        return
    }
    clearing.value = true
    try {
        const resp = await http.post('/manage/collect/clear', { id: clearSource.value.id, password: password.value })
        ElMessage[resp.data.code === 0 ? 'success' : 'error']({ message: resp.data.msg })
        if (resp.data.code === 0) clearV.value = false
    } finally {
        clearing.value = false
        password.value = ''
    }
}

</script>

<style scoped>
.row-tag {
  margin-left: 6px;
}
.op-dropdown {
  margin: 0 12px;
  vertical-align: middle;
}
.clear-stat {
  min-height: 60px;
  margin-bottom: 12px;
  font-size: 13px;
  line-height: 1.8;
}
.muted {
  color: var(--el-text-color-placeholder);
}
.clear-warn {
  color: var(--el-color-danger);
}
</style>
