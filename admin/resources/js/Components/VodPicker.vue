<template>
  <AdminDialog v-model="visible" width="760px" title="选择视频" confirm-text="关闭" @confirm="visible = false">
    <div class="picker-bar">
      <el-input v-model="keyword" clearable placeholder="片名或视频ID" @keyup.enter="load(1)" @clear="load(1)">
        <template #append><el-button :icon="Search" @click="load(1)"/></template>
      </el-input>
    </div>
    <el-table v-loading="loading" stripe :data="list" size="small" table-layout="auto" empty-text="此分类方案中没有符合的视频"
              class="picker-table" @row-click="pick">
      <el-table-column prop="id" label="ID" width="80"/>
      <el-table-column label="封面" width="64">
        <template #default="{ row }"><el-image :src="row.picture" fit="cover" class="pic"/></template>
      </el-table-column>
      <el-table-column prop="name" label="名称" min-width="160"/>
      <el-table-column prop="cName" label="分类" width="100"/>
      <el-table-column prop="year" label="年份" width="70"/>
      <el-table-column prop="remarks" label="备注" width="110" show-overflow-tooltip/>
      <el-table-column label="" width="80" align="center">
        <template #default="{ row }">
          <el-tag v-if="row.id === selected" type="success" size="small" disable-transitions>已绑定</el-tag>
          <el-button v-else size="small" type="primary" link>选择</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination v-if="page.total > page.pageSize" class="picker-pages" size="small" background layout="prev, pager, next, total"
                   :total="page.total" :page-size="page.pageSize" :current-page="page.current" @current-change="load"/>
  </AdminDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import AdminDialog from './AdminDialog.vue'
import { http } from '../utils/http'

export type PickedVod = { id: number; name: string; picture: string }

// 在分类方案中挑选一部视频 (海报绑定等): 点一行即选中并关闭
const props = defineProps<{ scheme: number; selected?: number }>()
const emit = defineEmits<{ select: [vod: PickedVod] }>()
const visible = defineModel<boolean>('visible', { required: true })

const keyword = ref('')
const loading = ref(false)
const list = ref<(PickedVod & { cName: string; year: string; remarks: string })[]>([])
const page = reactive({ current: 1, pageSize: 10, total: 0 })

const load = async (current = 1) => {
  loading.value = true
  try {
    const resp = await http.get('/manage/banner/vods', { params: { scheme: props.scheme, keyword: keyword.value, current } })
    if (resp.data.code === 0) {
      list.value = resp.data.data.list ?? []
      Object.assign(page, resp.data.data.page)
    } else {
      ElMessage.error(resp.data.msg)
    }
  } finally {
    loading.value = false
  }
}
// 每次打开时重新查询
watch(visible, (v) => { if (v) load(1) })

const pick = (row: PickedVod) => {
  emit('select', { id: row.id, name: row.name, picture: row.picture })
  visible.value = false
}
</script>

<style scoped>
.picker-bar {
  margin-bottom: 10px;
}
.picker-table :deep(.el-table__row) {
  cursor: pointer;
}
.pic {
  width: 36px;
  height: 48px;
  border-radius: 3px;
}
.picker-pages {
  margin-top: 10px;
  justify-content: flex-end;
}
</style>
