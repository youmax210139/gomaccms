<template>
  <div>
    <p v-if="page.props.errors?.form && !dialog" class="form-error">{{ page.props.errors.form }}</p>
    <TableToolbar>
      <el-button type="primary" :icon="Plus" @click="openAdd">添加</el-button>
      <el-button :icon="Delete" :disabled="selected.length === 0" @click="delAdmins(selected)">删除</el-button>
    </TableToolbar>
    <el-table stripe :data="admins" size="default" table-layout="auto" row-key="ID" empty-text="暂无管理员"
              @selection-change="(rows: Admin[]) => selected = rows">
      <el-table-column type="selection" width="48" :selectable="(row: Admin) => row.ID !== founderId && row.ID !== me"/>
      <el-table-column prop="ID" label="编号" align="center" width="90"/>
      <el-table-column label="名称" min-width="160">
        <template #default="{ row }">
          {{ row.userName }}<span v-if="row.nickName && row.nickName !== row.userName" class="sub">({{ row.nickName }})</span>
          <el-tag v-if="row.ID === founderId" size="small" type="danger" class="tag">创始人</el-tag>
          <el-tag v-else-if="row.ID === me" size="small" class="tag">当前</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" align="center" width="100">
        <template #default="{ row }">
          <el-switch :model-value="row.status === 1" inline-prompt active-text="启用" inactive-text="停用"
                     :disabled="row.ID === founderId || row.ID === me"
                     @change="(v: string | number | boolean) => changeStatus(row, v ? 1 : 0)"/>
        </template>
      </el-table-column>
      <el-table-column label="上次登录时间" align="center" width="180">
        <template #default="{ row }">{{ fmtTime(row.lastLoginAt) }}</template>
      </el-table-column>
      <el-table-column prop="lastLoginIp" label="上次登录IP" align="center" width="150"/>
      <el-table-column prop="loginCount" label="登录次数" align="center" width="100"/>
      <el-table-column label="操作" align="center" width="160">
        <template #default="{ row }">
          <el-button size="small" :disabled="row.ID === founderId && !isFounder" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" :disabled="row.ID === founderId || row.ID === me" @click="delAdmins([row])">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog v-model="dialog" width="760px" :title="form.id ? '编辑管理员' : '添加管理员'" :loading="form.processing" @confirm="save">
      <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
      <el-form :model="form" label-width="90px">
        <el-form-item label="账号" required>
          <el-input v-model="form.userName" maxlength="30" placeholder="字母、数字与下划线, 3-30 个字符"/>
        </el-form-item>
        <el-form-item label="密码" :required="!form.id">
          <el-input v-model="form.password" type="password" show-password autocomplete="new-password"
                    :placeholder="form.id ? '不修改请留空' : '至少 6 个字符'"/>
        </el-form-item>
        <el-form-item label="昵称">
          <el-input v-model="form.nickName" maxlength="60" placeholder="留空时与账号相同"/>
        </el-form-item>
        <el-form-item label="Email">
          <el-input v-model="form.email" maxlength="100"/>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" inline-prompt active-text="启用" inactive-text="停用"
                     :disabled="form.id === founderId || form.id === me"/>
        </el-form-item>
        <el-form-item label="权限">
          <p v-if="form.id === founderId" class="field-tip">创始管理员拥有全部权限</p>
          <div v-else class="perm-tree">
            <p class="field-tip">首页、个人信息不需要授权; 只勾选页面不勾选「--」开头的操作时, 只能查看与编辑, 不能执行这些操作。</p>
            <div v-for="g in permissions" :key="g.name" class="perm-group">
              <el-checkbox :model-value="groupState(g) === 'all'" :indeterminate="groupState(g) === 'some'"
                           :disabled="!groupEditable(g)" class="perm-group-name" @change="(v: any) => toggleGroup(g, Boolean(v))">{{ g.name }}</el-checkbox>
              <div class="perm-items">
                <template v-for="p in g.items" :key="p.key">
                  <el-checkbox :model-value="has(p.key)" :disabled="!grantable(p.key)" @change="(v: any) => toggle(p, Boolean(v))">{{ p.name }}</el-checkbox>
                  <el-checkbox v-for="a in p.actions ?? []" :key="a.key" :model-value="has(a.key)" :disabled="!grantable(a.key)"
                               @change="(v: any) => toggleAction(p, a, Boolean(v))">--{{ a.name }}</el-checkbox>
                </template>
              </div>
            </div>
          </div>
        </el-form-item>
      </el-form>
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
import { formatDateTime } from '../../utils/format'

type Admin = {
    ID: number; userName: string; nickName: string; email: string; status: number
    lastLoginAt: string | null; lastLoginIp: string; loginCount: number; permissions: string[] | null
}
type Perm = { key: string; name: string; actions?: Perm[] }
type PermGroup = { name: string; items: Perm[] }

const props = defineProps<{ admins: Admin[] | null; permissions: PermGroup[]; founderId: number; isFounder: boolean }>()
const page = usePage<{ errors?: { form?: string }; currentUser: { id: number; permissions: string[] | null } }>()

defineOptions({ layout: AdminLayout })

const admins = computed(() => props.admins ?? [])
const me = computed(() => page.props.currentUser?.id)
const selected = ref<Admin[]>([])

const fmtTime = (t: string | null) => formatDateTime(t, '-')

const dialog = ref(false)
const form = useForm({ id: 0, userName: '', password: '', nickName: '', email: '', status: 1, permissions: [] as string[] })
const openAdd = () => {
    Object.assign(form, { id: 0, userName: '', password: '', nickName: '', email: '', status: 1, permissions: [] })
    dialog.value = true
}
const openEdit = (a: Admin) => {
    Object.assign(form, { id: a.ID, userName: a.userName, password: '', nickName: a.nickName, email: a.email, status: a.status,
        permissions: [...(a.permissions ?? [])] })
    dialog.value = true
}
const save = () => {
    form.post('/manage/admin/save', {
        preserveScroll: true,
        onSuccess: () => { if (!page.props.errors?.form) dialog.value = false },
    })
}

// 权限勾选: 非创始管理员只能授予自己拥有的权限
const grantable = (key: string) => props.isFounder || !!page.props.currentUser?.permissions?.includes(key)
const has = (key: string) => form.permissions.includes(key)
const set = (key: string, on: boolean) => {
    if (!grantable(key) || has(key) === on) return
    form.permissions = on ? [...form.permissions, key] : form.permissions.filter((k) => k !== key)
}
// 取消页面时同时取消其操作; 勾选操作时同时勾选其页面
const toggle = (p: Perm, on: boolean) => {
    set(p.key, on)
    if (!on) (p.actions ?? []).forEach((a) => set(a.key, false))
}
const toggleAction = (p: Perm, a: Perm, on: boolean) => {
    set(a.key, on)
    if (on) set(p.key, true)
}
const groupKeys = (g: PermGroup) => g.items.flatMap((p) => [p.key, ...(p.actions ?? []).map((a) => a.key)])
const groupState = (g: PermGroup) => {
    const keys = groupKeys(g)
    const n = keys.filter(has).length
    return n === 0 ? 'none' : n === keys.length ? 'all' : 'some'
}
const groupEditable = (g: PermGroup) => groupKeys(g).some(grantable)
const toggleGroup = (g: PermGroup, on: boolean) => groupKeys(g).forEach((k) => set(k, on))

const changeStatus = (a: Admin, status: number) => {
    router.post('/manage/admin/state', { id: a.ID, status }, { preserveScroll: true })
}
const delAdmins = async (rows: Admin[]) => {
    const names = rows.map((r) => r.userName).join('、')
    if (!await confirmAction(`确认删除管理员「${names}」? 删除后无法恢复。`)) return
    router.post('/manage/admin/del', { ids: rows.map((r) => r.ID) }, { preserveScroll: true })
}
</script>

<style scoped>
.sub {
  margin-left: 4px;
  color: #909399;
}
.tag {
  margin-left: 6px;
}
.field-tip {
  margin: 0;
  color: #909399;
  font-size: 12px;
  line-height: 1.6;
}
.perm-tree {
  width: 100%;
}
.perm-group {
  display: flex;
  gap: 12px;
  padding: 6px 0;
  border-bottom: 1px dashed var(--el-border-color-lighter);
}
.perm-group-name {
  flex-shrink: 0;
  width: 90px;
  font-weight: 600;
}
.perm-items {
  display: flex;
  flex-wrap: wrap;
  column-gap: 4px;
}
</style>
