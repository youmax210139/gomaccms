<template>
  <div>
    <TableToolbar>
      <el-dropdown @command="clearLogs">
        <el-button :icon="Delete">清理日志</el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item :command="30">清理 30 天前的日志</el-dropdown-item>
            <el-dropdown-item :command="90">清理 90 天前的日志</el-dropdown-item>
            <el-dropdown-item :command="0" divided>清空全部日志</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <template #right>
        <el-select v-model="filter.userName" placeholder="账号" clearable filterable style="width: 140px">
          <el-option v-for="n in userNames" :key="n" :label="n" :value="n"/>
        </el-select>
        <el-select v-model="filter.success" placeholder="结果" clearable style="width: 100px">
          <el-option label="成功" value="1"/>
          <el-option label="失败" value="0"/>
        </el-select>
        <el-input v-model="filter.keyword" placeholder="操作 / 路径 / IP" clearable style="width: 200px" @keyup.enter="search"/>
        <el-button type="primary" :icon="Search" @click="search">查询</el-button>
      </template>
    </TableToolbar>
    <el-table stripe :data="logs" size="default" table-layout="auto" row-key="id" empty-text="暂无日志">
      <el-table-column prop="id" label="编号" align="center" width="90"/>
      <el-table-column label="时间" align="center" width="180">
        <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column prop="userName" label="账号" width="120"/>
      <el-table-column prop="action" label="操作" min-width="180"/>
      <el-table-column label="结果" align="center" width="80">
        <template #default="{ row }">
          <el-tag :type="row.success ? 'success' : 'danger'" size="small" disable-transitions>{{ row.success ? '成功' : '失败' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="ip" label="IP" align="center" width="140"/>
      <el-table-column label="请求" min-width="260" show-overflow-tooltip>
        <template #default="{ row }"><span class="req">{{ row.method }} {{ row.path }}</span> <span class="sub">{{ row.params }}</span></template>
      </el-table-column>
    </el-table>
    <div class="pagination">
      <el-pagination :page-sizes="[10, 20, 50, 100, 500]" background layout="prev, pager, next, sizes, total, jumper"
                     :total="query.paging.total" :page-size="query.paging.pageSize" :current-page="query.paging.current"
                     @change="(current: number, pageSize: number) => visit({ current, pageSize })"/>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { router } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import { Delete, Search } from '@element-plus/icons-vue'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import TableToolbar from '../../Components/TableToolbar.vue'
import { confirmAction } from '../../utils/confirm'
import { formatDateTime } from '../../utils/format'

type Log = { id: number; createdAt: string; userName: string; ip: string; method: string; path: string; action: string; params: string; success: boolean }
type Query = { userName: string; keyword: string; success: string; paging: { current: number; pageSize: number; total: number } }

const props = defineProps<{ logs: Log[] | null; userNames: string[] | null; query: Query; flash?: string }>()

defineOptions({ layout: AdminLayout })

const logs = computed(() => props.logs ?? [])
const userNames = computed(() => props.userNames ?? [])

const filter = reactive({ userName: '', keyword: '', success: '' })
watch(() => props.query, (q) => Object.assign(filter, { userName: q.userName, keyword: q.keyword, success: q.success }), { immediate: true })
watch(() => props.flash, (msg) => { if (msg) ElMessage.success(msg) }, { immediate: true })

const visit = (q: { current?: number; pageSize?: number }) => {
    router.get('/manage/admin/log/list', {
        userName: filter.userName || undefined, keyword: filter.keyword || undefined, success: filter.success || undefined,
        current: q.current ?? props.query.paging.current, pageSize: q.pageSize ?? props.query.paging.pageSize,
    }, { preserveState: true, preserveScroll: true })
}
const search = () => visit({ current: 1 })

const clearLogs = async (days: number) => {
    if (!await confirmAction(days ? `确认清理 ${days} 天前的操作日志?` : '确认清空全部操作日志? 清空后无法恢复。')) return
    router.post('/manage/admin/log/clear', { days })
}
</script>

<style scoped>
.req {
  font-family: monospace;
}
.sub {
  color: #909399;
  font-size: 12px;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>
