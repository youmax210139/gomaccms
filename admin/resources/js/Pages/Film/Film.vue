<template>
  <div>
    <p v-if="pageErr" class="form-error">{{ pageErr }}</p>
    <div class="params_form">
      <el-form class="cus_form">
        <el-form-item>
          <el-input v-model="params.name" placeholder="片名搜素" :suffix-icon="Search"/>
        </el-form-item>
        <el-form-item>
          <el-select v-model="schemeId" @change="changeScheme" placeholder="分类方案">
            <el-option v-for="s in options.schemes ?? []" :key="s.id" :label="s.name" :value="s.id"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="classId" @change="changeClass" placeholder="视频分类">
            <el-option v-for="item in options.class" :key="item.id" :label="item.name" :value="item.id"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="params.plot" placeholder="剧情筛选">
            <el-option v-for="item in tagOptions.Plot" :key="item.Value" :label="item.Name" :value="item.Value"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="params.area" placeholder="地区筛选">
            <el-option v-for="item in tagOptions.Area" :key="item.Value" :label="item.Name" :value="item.Value"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="params.language" placeholder="语言筛选">
            <el-option v-for="item in tagOptions.Language" :key="item.Value" :label="item.Name" :value="item.Value"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="params.year" placeholder="上映年份">
            <el-option v-for="item in options.year" :key="item.Value" :label="item.Name" :value="item.Value"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="params.remarks" placeholder="更新状态">
            <el-option v-for="item in options.remarks" :key="item.Value" :label="item.Name" :value="item.Value"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="params.status" placeholder="审核状态" clearable>
            <el-option label="已审核" value="1"/>
            <el-option label="未审核" value="0"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="params.level" placeholder="选择推荐" clearable>
            <el-option label="未推荐" value="0"/>
            <el-option v-for="n in 9" :key="n" :label="`推荐 ${n}`" :value="String(n)"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="params.lock" placeholder="选择锁定" clearable>
            <el-option label="已锁定" value="1"/>
            <el-option label="未锁定" value="0"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="params.player" placeholder="选择播放器" clearable>
            <el-option v-for="p in options.players ?? []" :key="p.code" :label="`${p.name} (${p.code})`" :value="p.code"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="params.picture" placeholder="选择图片" clearable>
            <el-option label="有图片" value="has"/>
            <el-option label="无图片" value="none"/>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-select v-model="params.sort" placeholder="选择排序" clearable>
            <el-option label="更新时间" value="time"/>
            <el-option label="添加时间" value="add"/>
            <el-option label="人气" value="hits"/>
            <el-option label="评分" value="score"/>
            <el-option label="编号" value="id"/>
          </el-select>
        </el-form-item>
        <el-form-item class="date-item">
          <el-date-picker v-model="dateGroup" value-format="YYYY-MM-DD HH:mm:ss" type="datetimerange" start-placeholder="起始时间" end-placeholder="终止时间"/>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="searchFilm">查询</el-button>
        </el-form-item>
      </el-form>
    </div>
    <TableToolbar>
      <el-button v-if="can('vod.add')" type="primary" :icon="CirclePlus" @click="router.visit('/manage/film/add')">添加</el-button>
      <el-button v-if="can('vod.delete')" type="danger" :icon="Delete" :disabled="!selected.length" @click="batchDel">删除</el-button>
      <template v-if="can('vod.batch')">
      <el-dropdown :disabled="!selected.length" @command="(v: number) => batchSet('level', v, v ? `推荐 ${v}` : '取消推荐')">
        <el-button :icon="Setting" :disabled="!selected.length">推荐</el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item :command="0">取消推荐</el-dropdown-item>
            <el-dropdown-item v-for="n in 9" :key="n" :command="n">推荐 {{ n }}</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-dropdown :disabled="!selected.length" @command="(v: number) => batchSet('status', v, v ? '设为已审核' : '设为未审核')">
        <el-button :icon="Setting" :disabled="!selected.length">审核</el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item :command="1">设为已审核</el-dropdown-item>
            <el-dropdown-item :command="0">设为未审核</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-dropdown :disabled="!selected.length" @command="(v: number) => batchSet('lock', v, v ? '锁定' : '解锁')">
        <el-button :icon="Setting" :disabled="!selected.length">锁定</el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item :command="1">锁定 (采集不再更新)</el-dropdown-item>
            <el-dropdown-item :command="0">解锁</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-button :icon="Setting" :disabled="!selected.length" @click="batchHits">人气</el-button>
      </template>
    </TableToolbar>
    <div class="content">
      <el-table stripe :data="list" style="width: 100%" size="default" table-layout="auto" max-height="calc(68vh - 20px)" row-key="mid"
                @selection-change="(rows: any[]) => (selected = rows)">
        <el-table-column type="selection" width="45"/>
        <el-table-column type="index" min-width="60" align="left" label="序号">
          <template #default="scope"><span>{{ serialNum(scope.$index) }}</span></template>
        </el-table-column>
        <el-table-column prop="mid" align="center" label="视频ID">
          <template #default="scope"><el-tag type="success" disable-transitions>{{ scope.row.mid }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="name" align="left" label="视频名称" show-overflow-tooltip class-name="col_name"/>
        <el-table-column align="center" label="推荐" width="80">
          <template #default="scope">
            <el-tag v-if="scope.row.level > 0" type="warning" disable-transitions>推荐 {{ scope.row.level }}</el-tag>
            <span v-else class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column align="center" label="审核" width="90">
          <template #default="scope">
            <el-switch :model-value="scope.row.status == 1" inline-prompt active-text="已审" inactive-text="未审"
                       @change="(v: string | number | boolean) => setOne(scope.row, 'status', v ? 1 : 0)"/>
          </template>
        </el-table-column>
        <el-table-column align="center" label="锁定" width="90">
          <template #default="scope">
            <el-switch :model-value="scope.row.lock == 1" inline-prompt active-text="锁定" inactive-text="未锁"
                       @change="(v: string | number | boolean) => setOne(scope.row, 'lock', v ? 1 : 0)"/>
          </template>
        </el-table-column>
        <el-table-column prop="cName" align="center" label="所属分类">
          <template #default="scope"><el-tag type="warning" disable-transitions>{{ scope.row.cName ? scope.row.cName : '暂无' }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="year" align="center" label="年份">
          <template #default="scope"><el-tag type="warning" disable-transitions>{{ scope.row.year }}</el-tag></template>
        </el-table-column>
        <el-table-column sortable prop="score" align="center" label="评分">
          <template #default="scope"><el-tag type="success" disable-transitions>{{ scope.row.score }}</el-tag></template>
        </el-table-column>
        <el-table-column sortable prop="hits" align="center" label="热度">
          <template #default="scope"><el-tag type="danger" disable-transitions>🔥{{ scope.row.hits }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="remarks" align="center" label="更新状态">
          <template #default="scope">
            <el-tag v-if="(scope.row.remarks + '').indexOf('更新') != -1" type="warning">{{ scope.row.remarks }}</el-tag>
            <el-tag v-else type="success">{{ scope.row.remarks }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column sortable prop="updateStamp" align="center" label="更新时间" min-width="130">
          <template #default="scope"><el-tag type="success" disable-transitions>{{ formatStamp(scope.row.updateStamp) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="操作" align="center" min-width="300">
          <template #default="scope">
            <el-button v-if="can('vod.edit')" size="small" @click="router.visit(`/manage/film/edit?id=${scope.row.mid}`)">编辑</el-button>
            <el-button v-if="can('vod.delete')" size="small" @click="delFilm(scope.row)">删除</el-button>
            <el-button size="small" @click="notReady('角色')">角色</el-button>
            <el-button size="small" @click="notReady('分集剧情')">分集剧情</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination :page-sizes="[10, 20, 50, 100, 500]" background layout="prev, pager, next, sizes, total, jumper"
                        :total="page.total" v-model:page-size="page.pageSize" v-model:current-page="page.current"
                        @change="reload" hide-on-single-page/>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CirclePlus, Delete, Search, Setting } from '@element-plus/icons-vue'
import { computed, reactive, ref, watch } from 'vue'
import { router, usePage } from '@inertiajs/vue3'
import { ElMessage, ElMessageBox } from 'element-plus'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import { confirmAction } from '../../utils/confirm'
import TableToolbar from '../../Components/TableToolbar.vue'
import { useCan } from '../../utils/menu'
import { formatStamp } from '../../utils/format'

const can = useCan()

const props = defineProps<{
  params: { name: string; schemeId: number; player: string; picture: string; sort: string; status: string; level: string; lock: string
    pid: number; cid: number; plot: string; area: string; language: string; year: number; remarks: string; paging: { current: number; pageSize: number; pageCount: number; total: number } }
  list: any[]
  options: { class: any[]; remarks: any[]; year: any[]; tags: Record<string, any>; schemes?: { id: number; name: string }[]
    players?: { code: string; name: string }[] }
}>()

defineOptions({ layout: AdminLayout })

// list/options reflect fresh server data on every reload() revisit, and reload() uses
// preserveState:true - this component instance is never remounted on revisit, so a plain
// snapshot (`const list = props.list`) would go stale forever after first render. Keep
// these as live bindings onto the props instead.
// Ported from the old SPA page: normalize year<=0 / score==0 display exactly as before.
// Also ported the old page's `resp.data.list ? ... : []` guard: GetFilmPage's underlying
// query can come back nil on a search-repo error (confirmed live: a MATCH()-based filter
// against a search table missing its FULLTEXT index throws, and the repo returns nil for
// the list while still returning valid, zeroed paging) - a bare props.list.map(...) would
// throw client-side (verified) whenever that happens, so guard for a null list here.
const list = computed(() => (props.list ?? []).map((item: any) => ({
    ...item,
    year: item.year <= 0 ? '未知' : item.year,
    score: item.score == 0 ? '暂无' : item.score,
})))
const options = computed(() => props.options)

// page backs v-model:page-size / v-model:current-page on <el-pagination>, so unlike list/
// options above it must stay a writable reactive object (a computed ref can't be v-model'd
// directly) while still tracking fresh paging data pushed in on revisit, hence the watch.
const page = reactive({ ...props.params.paging })
watch(() => props.params.paging, (p) => Object.assign(page, p))

// params/classId/dateGroup/tagOptions/pid/cid are local filter-input UI state: they seed
// once from the initial params but from then on are only ever driven by user interaction
// (typing, selecting, the category cascade), never by a later revisit's props, so they
// intentionally do NOT re-sync.
const params = reactive({
    name: props.params.name, plot: props.params.plot, area: props.params.area,
    language: props.params.language, year: props.params.year, remarks: props.params.remarks,
    player: props.params.player, picture: props.params.picture, sort: props.params.sort,
    status: props.params.status, level: props.params.level, lock: props.params.lock,
})
const selected = ref<any[]>([])
const classId = ref(props.params.cid || props.params.pid || 0)
const schemeId = ref(props.params.schemeId || 1)
const dateGroup = ref<string[]>([])
const tagOptions = reactive<{ Plot: any[]; Area: any[]; Language: any[] }>({ Plot: [], Area: [], Language: [] })
let pid = props.params.pid
let cid = props.params.cid

const serialNum = (index: number) => (page.current - 1) * page.pageSize + index + 1


const changeClass = (value: number) => {
    const c = options.value.class.find((item: any) => item.id === value)
    if (!c) return
    if (c.pid <= 0) {
        pid = c.id
        cid = 0
        return
    }
    if (c.pid === pid) {
        cid = c.id
        return
    }
    pid = c.pid
    cid = c.id
    const t = options.value.tags[String(c.pid == 0 ? c.id : c.pid)]
    tagOptions.Plot = t ? t['Plot'] : []
    tagOptions.Area = t ? t['Area'] : []
    tagOptions.Language = t ? t['Language'] : []
    params.plot = ''
    params.area = ''
    params.language = ''
}

// 切换分类方案: 分类选项换成该方案的分类, 清除已选分类并重新查询
const changeScheme = () => {
    classId.value = 0
    pid = 0
    cid = 0
    searchFilm()
}

const reload = () => {
    const [beginTime, endTime] = dateGroup.value?.length === 2 ? dateGroup.value : ['', '']
    router.get('/manage/film/search/list', {
        name: params.name, schemeId: schemeId.value, pid, cid, plot: params.plot, area: params.area, language: params.language,
        year: params.year, remarks: params.remarks, player: params.player, picture: params.picture, sort: params.sort,
        status: params.status, level: params.level, lock: params.lock, beginTime, endTime,
        current: page.current, pageSize: page.pageSize,
    }, { preserveState: true })
}

const searchFilm = () => {
    page.current = 1
    reload()
}

// 角色 / 分集剧情 尚未实现, 先占位
const notReady = (name: string) => ElMessage.info(`「${name}」功能开发中`)

const delFilm = async (row: any) => {
    if (!await confirmAction(`确认删除视频「${row.name}」?`)) return
    router.get('/manage/film/search/del', { id: row.ID }, { preserveScroll: true })
}

const batchDel = async () => {
    if (!await confirmAction(`确认删除选中的 ${selected.value.length} 部视频? 视频的分类与来源记录会一并删除。`)) return
    router.post('/manage/film/search/batch/del', { ids: selected.value.map((r) => r.mid) }, { preserveScroll: true })
}

// ---- 批量设置推荐 / 审核 / 锁定 / 人气
const inertiaPage = usePage<{ errors?: { form?: string } }>()
const pageErr = computed(() => inertiaPage.props.errors?.form)
const postUpdate = (ids: number[], field: string, value: number) => {
    router.post('/manage/film/search/batch/update', { ids, field, value }, { preserveScroll: true, preserveState: true })
}
const batchSet = async (field: string, value: number, label: string) => {
    if (!await confirmAction(`确认将选中的 ${selected.value.length} 部视频${label}?`)) return
    postUpdate(selected.value.map((r) => r.mid), field, value)
}
const setOne = (row: any, field: string, value: number) => postUpdate([row.mid], field, value)
const batchHits = async () => {
    try {
        const { value } = await ElMessageBox.prompt(`设置选中的 ${selected.value.length} 部视频的人气`, '人气', {
            inputPattern: /^\d{1,12}$/, inputErrorMessage: '请输入非负整数', confirmButtonText: '确定', cancelButtonText: '取消',
        })
        postUpdate(selected.value.map((r) => r.mid), 'hits', Number(value))
    } catch { /* 取消 */ }
}
</script>

<style scoped>
.params_form {
  background: #fff;
  margin-bottom: 20px;
  padding: 10px 20px;
}
/* 参照苹果 CMS 的筛选区: 等宽的格子排列 */
.cus_form {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 0 12px;
}
.cus_form :deep(.el-form-item) {
  margin-bottom: 12px;
  margin-right: 0;
}
.cus_form :deep(.el-select),
.cus_form :deep(.el-input) {
  width: 100%;
}
.cus_form .date-item {
  grid-column: span 2;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.muted {
  color: var(--el-text-color-placeholder);
}
</style>
