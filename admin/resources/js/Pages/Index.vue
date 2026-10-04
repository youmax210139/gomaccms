<template>
  <div>
    <h2>後台管理中心</h2>
    <p class="welcome">歡迎回來,{{ currentUser.nickName }}。</p>

    <el-card shadow="never" class="recent">
      <template #header>
        <div class="card-head">
          <span>最近采集</span>
          <el-link type="primary" underline="never" @click="router.get('/manage/collect/list')">采集接口 →</el-link>
        </div>
      </template>
      <el-table stripe :data="recentCollects ?? []" size="small" empty-text="还没有采集记录">
        <el-table-column prop="sourceName" label="采集接口" min-width="120"/>
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
        <el-table-column label="结束时间" width="170">
          <template #default="{ row }">{{ formatDateTime(row.finishedAt) }}</template>
        </el-table-column>
        <el-table-column prop="message" label="说明" min-width="200" show-overflow-tooltip/>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { router } from '@inertiajs/vue3'
import AdminLayout from '../Layouts/AdminLayout.vue'
import { collectStates } from '../utils/collectTasks'
import { formatDateTime } from '../utils/format'

defineProps<{
  currentUser: { nickName: string }
  recentCollects: any[] | null
}>()

defineOptions({ layout: AdminLayout })
</script>

<style scoped>
.welcome {
  color: var(--el-text-color-regular);
  margin-bottom: 16px;
}
.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
