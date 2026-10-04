<template>
  <div>
    <p v-if="page.props.errors?.form && !dialog" class="form-error">{{ page.props.errors.form }}</p>
    <p class="tip">游客为未登录的访客 (前台目前没有会员登录, 所有访客都套用游客组的权限); 默认会员为新会员与会员组到期后的会员组。删除会员组时, 其会员改为默认会员。</p>
    <TableToolbar>
      <el-button type="primary" :icon="Plus" @click="openAdd">添加</el-button>
      <el-button :icon="Delete" :disabled="selected.length === 0" @click="delGroups(selected)">删除</el-button>
    </TableToolbar>
    <el-table stripe :data="groups" size="default" table-layout="auto" row-key="id" empty-text="暂无会员组"
              @selection-change="(rows: Group[]) => selected = rows">
      <el-table-column type="selection" width="48" :selectable="(row: Group) => !row.builtin"/>
      <el-table-column prop="id" label="编号" align="center" width="90"/>
      <el-table-column label="名称" min-width="140">
        <template #default="{ row }">
          {{ row.name }}<el-tag v-if="row.builtin" size="small" type="info" class="tag">内置</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" align="center" width="100">
        <template #default="{ row }">
          <el-switch v-if="!row.builtin" :model-value="row.status" inline-prompt active-text="启用" inactive-text="停用"
                     @change="(v: string | number | boolean) => changeStatus(row, Boolean(v))"/>
          <span v-else class="sub">启用</span>
        </template>
      </el-table-column>
      <el-table-column prop="priceDay" label="包天" align="center" width="90"/>
      <el-table-column prop="priceWeek" label="包周" align="center" width="90"/>
      <el-table-column prop="priceMonth" label="包月" align="center" width="90"/>
      <el-table-column prop="priceYear" label="包年" align="center" width="90"/>
      <el-table-column label="会员数" align="center" width="90">
        <template #default="{ row }">{{ row.id === 1 ? '-' : row.members }}</template>
      </el-table-column>
      <el-table-column label="操作" align="center" width="160">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-button v-if="!row.builtin" size="small" @click="delGroups([row])">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog v-model="dialog" width="860px" :title="form.id ? '编辑会员组' : '添加会员组'" :loading="form.processing" @confirm="save">
      <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
      <el-tabs v-model="tab">
        <el-tab-pane label="基本设置" name="basic">
          <el-form :model="form" label-width="90px">
            <el-form-item label="名称" required>
              <el-input v-model="form.name" maxlength="30"/>
            </el-form-item>
            <el-form-item label="状态">
              <el-switch v-model="form.status" inline-prompt active-text="启用" inactive-text="停用" :disabled="builtin"/>
            </el-form-item>
            <el-form-item label="价格">
              <div class="prices">
                <span>包天</span><el-input-number v-model="form.priceDay" :min="0" controls-position="right"/>
                <span>包周</span><el-input-number v-model="form.priceWeek" :min="0" controls-position="right"/>
                <span>包月</span><el-input-number v-model="form.priceMonth" :min="0" controls-position="right"/>
                <span>包年</span><el-input-number v-model="form.priceYear" :min="0" controls-position="right"/>
              </div>
              <p class="field-tip">单位为积分, 供会员购买会员组使用 (前台购买功能尚未开放)</p>
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="form.remark" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" maxlength="255" show-word-limit/>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="分类权限" name="perm">
          <p class="field-tip">勾选该会员组在各分类可访问的页面; 视频的一级与二级分类都需要允许。下载页与试看暂未在前台生效。</p>
          <el-tabs v-if="schemes.length > 1" v-model="schemeTab" class="scheme-tabs">
            <el-tab-pane v-for="s in schemes" :key="s.id" :label="s.name" :name="String(s.id)"/>
          </el-tabs>
          <el-table stripe :data="currentCategories" size="small" max-height="420" empty-text="该方案暂无视频分类">
            <el-table-column label="分类" min-width="180">
              <template #default="{ row }">
                <span :style="{ paddingLeft: `${row.level * 18}px` }">{{ row.level ? '├ ' : '' }}{{ row.name }}</span>
              </template>
            </el-table-column>
            <el-table-column v-for="p in perms" :key="p" align="center" width="96">
              <template #header>
                <el-checkbox :model-value="columnState(p) === 'all'" :indeterminate="columnState(p) === 'some'"
                             @change="(v: any) => toggleColumn(p, Boolean(v))">{{ permLabels[p] }}</el-checkbox>
              </template>
              <template #default="{ row }">
                <el-checkbox :model-value="allowed(row.id, p)" @change="(v: any) => setPerm(row.id, p, Boolean(v))"/>
              </template>
            </el-table-column>
            <el-table-column label="整行" align="center" width="70">
              <template #default="{ row }">
                <el-checkbox :model-value="perms.every((p) => allowed(row.id, p))"
                             :indeterminate="perms.some((p) => allowed(row.id, p)) && !perms.every((p) => allowed(row.id, p))"
                             @change="(v: any) => perms.forEach((p) => setPerm(row.id, p, Boolean(v)))"/>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </AdminDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { router, useForm, usePage } from '@inertiajs/vue3'
import { Delete, Plus } from '@element-plus/icons-vue'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import AdminDialog from '../../Components/AdminDialog.vue'
import TableToolbar from '../../Components/TableToolbar.vue'
import { confirmAction } from '../../utils/confirm'

type Group = {
    id: number; name: string; status: boolean; priceDay: number; priceWeek: number; priceMonth: number; priceYear: number
    remark: string; permissions: Record<string, string[]> | null; builtin: boolean; members: number
}
type Scheme = { id: number; name: string; categories: { id: number; name: string; level: number }[] | null }

const props = defineProps<{ groups: Group[] | null; schemes: Scheme[] | null; perms: string[] }>()
const page = usePage<{ errors?: { form?: string } }>()

defineOptions({ layout: AdminLayout })

const permLabels: Record<string, string> = { list: '列表页', detail: '内容页', play: '播放页', down: '下载页', trial: '试看' }

const groups = computed(() => props.groups ?? [])
const schemes = computed(() => props.schemes ?? [])
const selected = ref<Group[]>([])

const dialog = ref(false)
const tab = ref('basic')
const schemeTab = ref('')
const form = useForm({
    id: 0, name: '', status: true, priceDay: 0, priceWeek: 0, priceMonth: 0, priceYear: 0, remark: '',
    permissions: {} as Record<string, string[]>,
})
const builtin = computed(() => form.id === 1 || form.id === 2)

// 没有设定的分类视为全部允许: 打开弹窗时为每个分类补上完整的权限, 保存时全部提交
const fullPermissions = (existing: Record<string, string[]> | null) => {
    const out: Record<string, string[]> = {}
    for (const s of schemes.value) {
        for (const c of s.categories ?? []) {
            out[c.id] = existing?.[c.id] ? [...existing[c.id]] : [...props.perms]
        }
    }
    return out
}
const open = (values: Partial<Group>) => {
    Object.assign(form, {
        id: 0, name: '', status: true, priceDay: 0, priceWeek: 0, priceMonth: 0, priceYear: 0, remark: '', ...values,
        permissions: fullPermissions(values.permissions ?? null),
    })
    tab.value = 'basic'
    schemeTab.value = String(schemes.value[0]?.id ?? '')
    dialog.value = true
}
const openAdd = () => open({})
const openEdit = (g: Group) => open(g)
const save = () => {
    form.transform((data) => {
        const { id, name, status, priceDay, priceWeek, priceMonth, priceYear, remark, permissions } = data
        return { id, name, status, priceDay, priceWeek, priceMonth, priceYear, remark, permissions }
    }).post('/manage/member/group/save', {
        preserveScroll: true,
        onSuccess: () => { if (!page.props.errors?.form) dialog.value = false },
    })
}

const currentCategories = computed(() =>
    (schemes.value.length > 1 ? schemes.value.find((s) => String(s.id) === schemeTab.value) : schemes.value[0])?.categories ?? [])
const allowed = (cid: number, p: string) => form.permissions[cid]?.includes(p) ?? true
const setPerm = (cid: number, p: string, on: boolean) => {
    const l = form.permissions[cid] ?? [...props.perms]
    form.permissions[cid] = on ? [...new Set([...l, p])] : l.filter((x) => x !== p)
}
const columnState = (p: string) => {
    const l = currentCategories.value
    const n = l.filter((c) => allowed(c.id, p)).length
    return n === 0 ? 'none' : n === l.length ? 'all' : 'some'
}
const toggleColumn = (p: string, on: boolean) => currentCategories.value.forEach((c) => setPerm(c.id, p, on))

const changeStatus = (g: Group, status: boolean) => {
    router.post('/manage/member/group/state', { id: g.id, status }, { preserveScroll: true })
}
const delGroups = async (rows: Group[]) => {
    const names = rows.map((r) => r.name).join('、')
    if (!await confirmAction(`确认删除会员组「${names}」? 其会员将改为默认会员。`)) return
    router.post('/manage/member/group/del', { ids: rows.map((r) => r.id) }, { preserveScroll: true })
}
</script>

<style scoped>
.tip {
  color: #909399;
  font-size: 13px;
  margin: 0 0 12px;
  line-height: 1.6;
}
.sub {
  color: #909399;
}
.tag {
  margin-left: 6px;
}
.prices {
  display: grid;
  grid-template-columns: auto 1fr auto 1fr;
  gap: 8px 10px;
  align-items: center;
}
.field-tip {
  width: 100%;
  margin: 0 0 8px;
  color: #909399;
  font-size: 12px;
  line-height: 1.6;
}
.scheme-tabs {
  margin-bottom: 4px;
}
</style>
