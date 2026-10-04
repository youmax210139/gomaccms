<template>
  <div>
    <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
    <el-tabs :model-value="String(scheme)" type="card" class="scheme-tabs" @tab-change="(name: string | number) => switchScheme(Number(name))">
      <el-tab-pane v-for="s in schemes" :key="s.id" :name="String(s.id)" :label="s.name"/>
    </el-tabs>
    <div class="scheme-bar">
      <span class="field-tip">使用中的域名:
        <template v-if="currentScheme?.domains?.length">
          <el-tag v-for="d in currentScheme.domains" :key="d" size="small" class="domain-tag" disable-transitions>{{ d }}</el-tag>
        </template>
        <span v-else>无 (可在「系统 → 站群管理」中为域名选用此方案)</span>
      </span>
    </div>
    <TableToolbar>
      <el-button type="primary" :icon="Plus" @click="openAdd(0)">添加</el-button>
      <el-button :icon="Edit" :disabled="selected.length !== 1" @click="openEdit(selected[0])">编辑</el-button>
      <el-button type="danger" :icon="Delete" :disabled="!selected.length" @click="batchDel">删除</el-button>
      <el-dropdown :disabled="!selected.length" @command="batchState">
        <el-button :icon="Setting" :disabled="!selected.length">状态</el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="on">设为启用</el-dropdown-item>
            <el-dropdown-item command="off">设为停用</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-button :icon="Switch" :disabled="!selected.length" @click="openTransfer">转移</el-button>
    </TableToolbar>

    <el-table stripe :data="rows" row-key="id" default-expand-all table-layout="auto" @selection-change="(rows: Row[]) => (selected = rows)">
      <el-table-column type="selection" width="45" />
      <el-table-column label="名称" min-width="240">
        <template #default="{ row }">
          <span class="class-id">{{ row.id }}、</span>
          <span>{{ row.name }}</span>
          <el-tag v-if="row.type == 1" class="class-count" type="danger" size="small" effect="dark" disable-transitions>{{ counts[row.id] ?? 0 }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="类型" align="center" width="90">
        <template #default="{ row }">{{ types[row.type] ?? '-' }}</template>
      </el-table-column>
      <el-table-column prop="slug" label="Slug" min-width="140" show-overflow-tooltip />
      <el-table-column label="级别" align="center" width="100">
        <template #default="{ row }">
          <el-tag :type="row.pid == 0 ? 'success' : 'warning'" disable-transitions>{{ row.pid == 0 ? '一级' : '二级' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" align="center" width="100">
        <template #default="{ row }">
          <el-switch :model-value="row.show" inline-prompt active-text="启用" inactive-text="停用" @change="(v: string | number | boolean) => changeShow(row, Boolean(v))" />
        </template>
      </el-table-column>
      <el-table-column label="排序" align="center" width="120">
        <template #default="{ row }">
          <el-input-number :model-value="row.sort ?? 0" :min="0" :controls="false" size="small" class="sort-input" @change="(v: number | undefined) => changeSort(row, v)" />
        </template>
      </el-table-column>
      <el-table-column label="操作" align="center" width="220">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" @click="delClass(row)">删除</el-button>
          <el-button v-if="row.pid == 0" size="small" @click="openAdd(row.id)">添加</el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog v-model="formDialog" width="600px" :title="form.id ? '编辑分类' : '添加分类'" :loading="form.processing" @confirm="saveClass">
      <el-form :model="form" label-width="130px">
        <el-form-item label="类型" required>
          <el-radio-group v-model="form.type" @change="form.pid = 0">
            <el-radio v-for="(label, value) in types" :key="value" :value="Number(value)">{{ label }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="上级分类">
          <el-select v-model="form.pid">
            <el-option :value="0" label="顶级分类" />
            <el-option v-for="c in parentOptions" :key="c.id" :value="c.id" :label="c.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="Slug" required>
          <el-input v-model="form.slug" maxlength="60" placeholder="前台 URL 标识, 唯一, 如 sci-fi" />
        </el-form-item>
        <el-form-item label="状态"><el-switch v-model="form.show" inline-prompt active-text="启用" inactive-text="停用" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" :step="1" step-strictly /> <span class="field-tip">数字越小越靠前</span></el-form-item>
        <LangTabs v-model:active="classLang" :languages="classLanguages" :i18n="form.i18n" :source="form" :fields="classFields" label-width="130px">
          <el-form-item label="名称" required>
            <el-input v-model="form.name" maxlength="60" placeholder="前台分类名称及页面 H1" />
          </el-form-item>
        </LangTabs>
      </el-form>
    </AdminDialog>

    <AdminDialog v-model="transferDialog" width="480px" title="转移视频" confirm-text="转移" :loading="transferForm.processing" @confirm="transfer">
      <p class="tip">将所选 {{ selected.length }} 个分类下的视频转移到目标分类 (一级分类会转移其下全部视频)。</p>
      <el-form label-width="80px">
        <el-form-item label="目标分类" required>
          <el-select v-model="transferForm.target" placeholder="选择二级分类" filterable>
            <el-option-group v-for="c in videoRows" :key="c.id" :label="c.name">
              <el-option v-for="sub in c.children ?? []" :key="sub.id" :value="sub.id" :label="sub.name" />
            </el-option-group>
          </el-select>
        </el-form-item>
      </el-form>
    </AdminDialog>
  </div>
</template>

<script setup lang="ts">
import { Delete, Edit, Plus, Setting, Switch } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'
import { router, useForm, usePage } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import { confirmAction } from '../../utils/confirm'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import AdminDialog from '../../Components/AdminDialog.vue'
import TableToolbar from '../../Components/TableToolbar.vue'
import LangTabs, { type LangField } from '../../Components/LangTabs.vue'
import { i18nOf, watchI18nLangs } from '../../utils/i18n'

type Row = {
  id: number; pid: number; type: number; name: string; slug: string; show: boolean; sort?: number
  i18n?: Record<string, ClassText> | null; children?: Row[] | null
}
type ClassText = { name: string }
type Scheme = { id: number; name: string; sort: number; defaultLang: string; langs: string[] | null; domains?: string[] | null }

const props = defineProps<{
  tree: { children: Row[] | null }
  counts: Record<string, number> | null
  types: Record<string, string>
  scheme: number
  schemes: Scheme[] | null
  classLanguages: { code: string; name: string }[] | null
}>()
const page = usePage<{ errors?: { form?: string } }>()

defineOptions({ layout: AdminLayout })

const rows = computed(() => props.tree?.children ?? [])
const counts = computed(() => props.counts ?? {})
const types = computed(() => props.types ?? {})
// ---- 分类方案
const schemes = computed(() => props.schemes ?? [])
const currentScheme = computed(() => schemes.value.find((s) => s.id == props.scheme) ?? null)
const switchScheme = (id: number) => {
    selected.value = []
    router.get('/manage/film/class/tree', { scheme: id })
}
const videoRows = computed(() => rows.value.filter((r) => r.type == 1))
const selected = ref<Row[]>([])
const ids = () => selected.value.map((r) => r.id)
const closeOnSuccess = (close: () => void) => ({ onSuccess: () => { if (!page.props.errors?.form) close() } })

// ---- 添加 / 编辑
const formDialog = ref(false)
// 当前方案启用的非原文语言, 每个语言一个页签
const classLanguages = computed(() => props.classLanguages ?? [])
const emptyText = (): ClassText => ({ name: '' })
const classFields: LangField[] = [
    { key: 'name', label: '名称', maxlength: 60 },
]
const emptyForm = () => ({
    id: 0, type: 1, pid: 0, name: '', slug: '', show: true, sort: 0,
    schemeId: 1, i18n: i18nOf(classLanguages.value, null, emptyText),
})
const form = useForm(emptyForm())
watchI18nLangs(classLanguages, () => form.i18n, emptyText)
const classLang = ref('')
// 上级分类只能选择同类型的一级分类 (不能选自己)
const parentOptions = computed(() => rows.value.filter((r) => r.type == form.type && r.id != form.id))

const openAdd = (pid: number) => {
    const parent = rows.value.find((r) => r.id == pid)
    Object.assign(form, emptyForm(), { pid, type: parent?.type ?? 1, schemeId: props.scheme })
    classLang.value = ''
    formDialog.value = true
}
const openEdit = (row: Row) => {
    const i18n = i18nOf(classLanguages.value, row.i18n, emptyText)
    Object.assign(form, {
        id: row.id, type: row.type, pid: row.pid, name: row.name, slug: row.slug, show: row.show, sort: row.sort ?? 0,
        schemeId: props.scheme, i18n,
    })
    classLang.value = ''
    formDialog.value = true
}
const saveClass = () => {
    if (!form.name.trim() || !form.slug.trim()) {
        ElMessage.warning('名称与 Slug 不能为空')
        return
    }
    form.post('/manage/film/class/save', closeOnSuccess(() => { formDialog.value = false }))
}

// ---- 行内修改
const changeShow = (row: Row, show: boolean) => {
    router.post('/manage/film/class/update', { id: row.id, show }, { preserveScroll: true })
}
const changeSort = (row: Row, sort: number | undefined) => {
    router.post('/manage/film/class/update', { id: row.id, sort: sort ?? 0 }, { preserveScroll: true })
}

// ---- 删除
const delClass = async (row: Row) => {
    const tip = row.pid == 0 && row.children?.length ? ', 其下的二级分类会一并删除' : ''
    if (!await confirmAction(`确认删除分类「${row.name}」${tip}?`)) return
    router.get('/manage/film/class/del', { id: row.id })
}
const batchDel = async () => {
    if (!await confirmAction(`确认删除所选 ${selected.value.length} 个分类?`)) return
    router.post('/manage/film/class/batch/del', { ids: ids() })
}

// ---- 批量状态
// el-dropdown-item 的 command 只接受 string / number / object, 用 'on' / 'off' 表示启用 / 停用
const batchState = async (command: string) => {
    const show = command === 'on'
    if (!await confirmAction(`确认将所选 ${selected.value.length} 个分类设为${show ? '启用' : '停用'}?`)) return
    router.post('/manage/film/class/batch/state', { ids: ids(), show })
}

// ---- 转移视频
const transferDialog = ref(false)
const transferForm = useForm({ ids: [] as number[], target: undefined as number | undefined })

const openTransfer = () => {
    Object.assign(transferForm, { ids: ids(), target: undefined })
    transferDialog.value = true
}
const transfer = async () => {
    if (!transferForm.target) {
        ElMessage.warning('请选择目标分类')
        return
    }
    if (!await confirmAction('确认转移所选分类下的视频?')) return
    transferForm.post('/manage/film/class/transfer', closeOnSuccess(() => { transferDialog.value = false }))
}
</script>

<style scoped>
.class-id {
  color: var(--el-text-color-secondary);
}
.class-count {
  margin-left: 8px;
}
.sort-input {
  width: 80px;
}
.field-tip {
  margin-left: 12px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.tip {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  margin: 0 0 16px;
}
.domain-tag {
  margin: 2px 4px 2px 0;
}
.scheme-tabs {
  margin-bottom: 4px;
}
.scheme-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}
</style>
