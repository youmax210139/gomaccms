<template>
  <div>
    <el-table stripe :data="st.schemes ?? []" size="default" table-layout="auto">
      <el-table-column label="分类方案" min-width="160">
        <template #default="{ row }">
          <div class="scheme-name">{{ row.name }}</div>
          <div class="hosts">{{ row.hosts?.length ? row.hosts.join(' · ') : '未配置推送域名' }}</div>
        </template>
      </el-table-column>
      <el-table-column label="Sitemap" min-width="230">
        <template #default="{ row }">
          <div>影片 <b>{{ row.sitemap.vodUrls.toLocaleString() }}</b> · 页面 {{ row.sitemap.pageUrls }} · 分片 {{ row.sitemap.shards }}</div>
          <div class="sub">
            {{ isZero(row.sitemap.builtAt) ? '尚未生成' : '生成于 ' + formatDateTime(row.sitemap.builtAt) }}
            <el-tag v-if="row.dirty" type="warning" size="small" disable-transitions>待重建</el-tag>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="IndexNow" min-width="190">
        <template #default="{ row }">
          <template v-if="st.indexnowEnabled">
            <div>待推送 <b>{{ row.indexnow.pending }}</b></div>
            <div class="sub">{{ isZero(row.indexnow.lastSubmit) ? '尚未提交' : `${formatDateTime(row.indexnow.lastSubmit)} 提交 ${row.indexnow.lastCount} 个` }}</div>
            <div v-if="row.indexnow.lastError" class="form-error err">{{ row.indexnow.lastError }}</div>
          </template>
          <span v-else class="sub">未启用</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="340">
        <template #default="{ row }">
          <el-button size="small" :loading="running('sitemap', row.id)" @click="act('sitemap', row.id)">刷新 Sitemap</el-button>
          <el-button size="small" @click="act('rss', row.id)">刷新 RSS</el-button>
          <el-button size="small" :disabled="!st.indexnowEnabled" :loading="running('indexnow', row.id)" @click="act('indexnow', row.id)">提交 IndexNow</el-button>
        </template>
      </el-table-column>
    </el-table>
    <p class="meta">
      每个分片最多 {{ st.chunkSize.toLocaleString() }} 个 URL ·
      {{ st.autoInterval ? `内容变动后每 ${st.autoInterval} 分钟自动重建` : '不自动重建' }} ·
      RSS 最新 {{ st.rssLimit }} 部, 缓存 {{ st.rssTtl }} 秒 ·
      IndexNow {{ st.indexnowEnabled ? '已启用, 推送到方案中网站开启的域名' : '未启用 (设置 INDEXNOW_ENABLED=true、INDEXNOW_KEY)' }}
    </p>

    <h3 class="title">任务</h3>
    <el-table stripe :data="st.jobs ?? []" size="default" table-layout="auto" empty-text="还没有任务">
      <el-table-column label="任务" min-width="180">
        <template #default="{ row }">{{ jobNames[row.type] ?? row.type }} · {{ schemeName(row.schemeId) }}</template>
      </el-table-column>
      <el-table-column label="状态" align="center" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTag[row.status]?.type ?? 'info'" size="small" disable-transitions>{{ statusTag[row.status]?.label ?? row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="进度" min-width="200">
        <template #default="{ row }">
          <el-progress v-if="row.total" :percentage="Math.min(100, Math.round(row.processed * 100 / row.total))"
                       :status="row.status === 'completed' ? 'success' : row.status === 'failed' ? 'exception' : undefined"/>
          <span v-else class="sub">-</span>
        </template>
      </el-table-column>
      <el-table-column label="数量" align="center" width="150">
        <template #default="{ row }">{{ row.processed.toLocaleString() }} / {{ row.total.toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="开始时间" min-width="160">
        <template #default="{ row }">{{ row.startedAt ? formatDateTime(row.startedAt) : '-' }}</template>
      </el-table-column>
      <el-table-column prop="error" label="错误" min-width="160" show-overflow-tooltip/>
    </el-table>
    <div class="pagination">
      <el-pagination :page-sizes="[10, 20, 50, 100]" background layout="prev, pager, next, sizes, total"
                     :total="st.jobsPage.total" v-model:page-size="st.jobsPage.pageSize" v-model:current-page="st.jobsPage.current"
                     @change="load"/>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import { http } from '../../utils/http'
import { formatDateTime } from '../../utils/format'

type Job = { id: string; type: string; schemeId: number; status: string; total: number; processed: number; error: string; startedAt: string | null }
type Scheme = {
  id: number; name: string; hosts: string[] | null; dirty: boolean
  sitemap: { vodUrls: number; pageUrls: number; shards: number; builtAt: string }
  indexnow: { pending: number; lastSubmit: string; lastCount: number; lastError: string }
}
type Page = { current: number; pageSize: number; total: number }
type Status = {
  schemes: Scheme[] | null; chunkSize: number; autoInterval: number; rssLimit: number; rssTtl: number; indexnowEnabled: boolean
  jobs: Job[] | null; jobsPage: Page
}

const props = defineProps<{ status: Status }>()
defineOptions({ layout: AdminLayout })

const st = reactive<Status>(props.status)
const jobNames: Record<string, string> = { sitemap: 'Sitemap 重建', indexnow: 'IndexNow 提交' }
const statusTag: Record<string, { label: string; type: 'primary' | 'success' | 'warning' | 'danger' | 'info' }> = {
  pending: { label: '等待中', type: 'info' },
  running: { label: '进行中', type: 'primary' },
  completed: { label: '已完成', type: 'success' },
  failed: { label: '失败', type: 'danger' },
}

const isZero = (t: string) => !t || t.startsWith('0001-')
// 方案 0 为改成按方案分开之前、一次处理全部方案的任务
const schemeName = (id: number) => (id === 0 ? '全部方案' : (st.schemes ?? []).find((s) => s.id === id)?.name ?? `方案 ${id}`)
const running = (type: string, schemeId: number) =>
  (st.jobs ?? []).some((j) => j.type === type && j.schemeId === schemeId && (j.status === 'running' || j.status === 'pending'))
const anyRunning = () => (st.jobs ?? []).some((j) => j.status === 'running' || j.status === 'pending')

// 有进行中的任务时每 2 秒更新一次状态
let timer: ReturnType<typeof setTimeout> | undefined
const load = async () => {
  clearTimeout(timer)
  try {
    // 带上任务列表的页码, 轮询时停留在当前页
    const resp = await http.get('/manage/publish/status', { params: { current: st.jobsPage.current, pageSize: st.jobsPage.pageSize } })
    if (resp.data.code === 0) Object.assign(st, resp.data.data)
  } catch { /* 下次再试 */ }
  if (anyRunning()) timer = setTimeout(load, 2000)
}
onBeforeUnmount(() => clearTimeout(timer))
if (anyRunning()) load()

const act = async (action: 'sitemap' | 'rss' | 'indexnow', schemeId: number) => {
  const resp = await http.post(`/manage/publish/${action}`, { schemeId })
  ElMessage[resp.data.code === 0 ? 'success' : 'error']({ message: resp.data.msg })
  st.jobsPage.current = 1 // 新任务在第一页
  load()
}
</script>

<style scoped>
.tip {
  color: #909399;
  font-size: 13px;
  margin: 0 0 12px;
  line-height: 1.6;
}
.tip code, .meta code {
  background: var(--el-fill-color-light);
  padding: 1px 6px;
  border-radius: 4px;
}
.scheme-name {
  font-weight: 600;
}
.hosts, .sub {
  color: #909399;
  font-size: 12px;
  margin-top: 2px;
}
.err {
  font-size: 12px;
  margin: 2px 0 0;
}
.meta {
  margin: 10px 0 24px;
  color: #606266;
  font-size: 13px;
  line-height: 1.8;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.title {
  margin: 0 0 10px;
  font-size: 16px;
}
</style>
