<template>
  <div>
    <p v-if="page.props.errors?.form && !dialog" class="form-error">{{ page.props.errors.form }}</p>
    <TableToolbar>
      <el-button type="primary" :icon="Plus" @click="openAdd">添加</el-button>
      <el-button :icon="Delete" :disabled="selected.length === 0" @click="delMembers(selected)">删除</el-button>
      <template #right>
        <el-select v-model="filter.groupId" placeholder="会员组" clearable style="width: 140px">
          <el-option v-for="g in memberGroups" :key="g.id" :label="g.name" :value="g.id"/>
        </el-select>
        <el-select v-model="filter.status" placeholder="状态" clearable style="width: 110px">
          <el-option label="启用" value="1"/>
          <el-option label="停用" value="0"/>
        </el-select>
        <el-input v-model="filter.keyword" placeholder="账号 / 昵称 / Email" clearable style="width: 200px" @keyup.enter="search"/>
        <el-button type="primary" :icon="Search" @click="search">查询</el-button>
      </template>
    </TableToolbar>
    <el-table stripe :data="members" size="default" table-layout="auto" row-key="id" empty-text="暂无会员"
              @selection-change="(rows: Member[]) => selected = rows">
      <el-table-column type="selection" width="48"/>
      <el-table-column prop="id" label="编号" align="center" width="90"/>
      <el-table-column label="账号" min-width="140">
        <template #default="{ row }">
          {{ row.userName }}<span v-if="row.nickName && row.nickName !== row.userName" class="sub">({{ row.nickName }})</span>
        </template>
      </el-table-column>
      <el-table-column label="会员组" align="center" width="120">
        <template #default="{ row }">{{ groupName(row.groupId) }}</template>
      </el-table-column>
      <el-table-column prop="points" label="积分" align="center" width="90"/>
      <el-table-column label="到期时间" align="center" width="180">
        <template #default="{ row }">
          <span :class="{ expired: row.expireAt && new Date(row.expireAt) < new Date() }">{{ row.expireAt ? fmtTime(row.expireAt) : '不过期' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" align="center" width="100">
        <template #default="{ row }">
          <el-switch :model-value="row.status" inline-prompt active-text="启用" inactive-text="停用"
                     @change="(v: string | number | boolean) => changeStatus(row, Boolean(v))"/>
        </template>
      </el-table-column>
      <el-table-column label="上次登录" align="center" width="180">
        <template #default="{ row }">{{ fmtTime(row.lastLoginAt) }}<div v-if="row.lastLoginIp" class="sub">{{ row.lastLoginIp }}</div></template>
      </el-table-column>
      <el-table-column prop="loginCount" label="登录次数" align="center" width="90"/>
      <el-table-column label="注册时间" align="center" width="180">
        <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" align="center" width="160">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" @click="delMembers([row])">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="pagination">
      <el-pagination :page-sizes="[10, 20, 50, 100, 500]" background layout="prev, pager, next, sizes, total, jumper"
                     :total="query.paging.total" :page-size="query.paging.pageSize" :current-page="query.paging.current"
                     @change="(current: number, pageSize: number) => visit({ current, pageSize })"/>
    </div>

    <AdminDialog v-model="dialog" width="560px" :title="form.id ? '编辑会员' : '添加会员'" :loading="form.processing" @confirm="save">
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
        <el-form-item label="会员组" required>
          <el-select v-model="form.groupId" style="width: 100%">
            <el-option v-for="g in memberGroups" :key="g.id" :label="g.name + (g.status ? '' : ' (已停用)')" :value="g.id"/>
          </el-select>
        </el-form-item>
        <el-form-item label="到期时间">
          <el-date-picker v-model="form.expireAt" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" placeholder="留空为不过期" style="width: 100%"/>
        </el-form-item>
        <el-form-item label="积分">
          <el-input-number v-model="form.points" :min="0"/>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" inline-prompt active-text="启用" inactive-text="停用"/>
        </el-form-item>
      </el-form>
    </AdminDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { router, useForm, usePage } from '@inertiajs/vue3'
import { Delete, Plus, Search } from '@element-plus/icons-vue'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import AdminDialog from '../../Components/AdminDialog.vue'
import TableToolbar from '../../Components/TableToolbar.vue'
import { confirmAction } from '../../utils/confirm'
import { formatDateTime } from '../../utils/format'

type Member = {
    id: number; userName: string; nickName: string; email: string; groupId: number; points: number
    expireAt: string | null; status: boolean; lastLoginAt: string | null; lastLoginIp: string; loginCount: number; createdAt: string
}
type Group = { id: number; name: string; status: boolean }
type Query = { groupId: number; status: string; keyword: string; paging: { current: number; pageSize: number; total: number } }

const props = defineProps<{ members: Member[] | null; groups: Group[] | null; query: Query }>()
const page = usePage<{ errors?: { form?: string } }>()

defineOptions({ layout: AdminLayout })

const members = computed(() => props.members ?? [])
// 会员不能属于游客组
const memberGroups = computed(() => (props.groups ?? []).filter((g) => g.id !== 1))
const groupName = (id: number) => props.groups?.find((g) => g.id === id)?.name ?? `#${id}`
const selected = ref<Member[]>([])

const fmtTime = (t: string | null) => formatDateTime(t, '-')

const filter = reactive({ groupId: undefined as number | undefined, status: '', keyword: '' })
watch(() => props.query, (q) => Object.assign(filter, { groupId: q.groupId || undefined, status: q.status, keyword: q.keyword }), { immediate: true })

const visit = (q: { current?: number; pageSize?: number }) => {
    router.get('/manage/member/list', {
        groupId: filter.groupId || undefined, status: filter.status || undefined, keyword: filter.keyword || undefined,
        current: q.current ?? props.query.paging.current, pageSize: q.pageSize ?? props.query.paging.pageSize,
    }, { preserveState: true, preserveScroll: true })
}
const search = () => visit({ current: 1 })

const dialog = ref(false)
const blank = { id: 0, userName: '', password: '', nickName: '', email: '', groupId: 2, points: 0, expireAt: '', status: true }
const form = useForm({ ...blank })
const openAdd = () => {
    Object.assign(form, blank)
    dialog.value = true
}
const openEdit = (m: Member) => {
    Object.assign(form, { ...blank, id: m.id, userName: m.userName, nickName: m.nickName, email: m.email, groupId: m.groupId,
        points: m.points, expireAt: formatDateTime(m.expireAt), status: m.status })
    dialog.value = true
}
const save = () => {
    form.transform((d) => ({ ...d, expireAt: d.expireAt ?? '' })).post('/manage/member/save', {
        preserveScroll: true,
        onSuccess: () => { if (!page.props.errors?.form) dialog.value = false },
    })
}
const changeStatus = (m: Member, status: boolean) => {
    router.post('/manage/member/state', { id: m.id, status }, { preserveScroll: true })
}
const delMembers = async (rows: Member[]) => {
    const names = rows.map((r) => r.userName).join('、')
    if (!await confirmAction(`确认删除会员「${names}」? 删除后无法恢复。`)) return
    router.post('/manage/member/del', { ids: rows.map((r) => r.id) }, { preserveScroll: true })
}
</script>

<style scoped>
.sub {
  margin-left: 4px;
  color: #909399;
  font-size: 12px;
}
.expired {
  color: #f56c6c;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>
