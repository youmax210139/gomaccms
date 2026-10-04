<template>
  <!-- 从「文件管理」已上传的图片中选择一张, 点击图片即选中并关闭弹窗 -->
  <AdminDialog v-model="visible" title="从图库选择" width="760px">
    <el-radio-group v-model="source" size="small" class="gallery-filter" @change="load(1)">
      <el-radio-button value="upload">手动上传</el-radio-button>
      <el-radio-button value="poster">视频封面</el-radio-button>
      <el-radio-button value="">全部</el-radio-button>
    </el-radio-group>
    <div v-loading="loading" class="gallery">
      <div v-for="img in list" :key="img.ID" :class="['gallery-item', { active: img.link === modelValue }]" :title="caption(img)" @click="pick(img.link)">
        <el-image :src="img.link" fit="cover" class="gallery-thumb">
          <template #error><div class="gallery-failed">文件不存在</div></template>
        </el-image>
        <div class="gallery-caption">{{ caption(img) }}</div>
      </div>
      <el-empty v-if="!loading && list.length === 0" description="图库暂无图片, 请先上传" class="gallery-empty"/>
    </div>
    <div class="gallery-pagination">
      <el-pagination background layout="prev, pager, next" :total="page.total" :page-size="page.pageSize"
                     :current-page="page.current" hide-on-single-page @current-change="load"/>
    </div>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
    </template>
  </AdminDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { http } from '../utils/http'
import AdminDialog from './AdminDialog.vue'

const visible = defineModel<boolean>('visible', { default: false })
defineProps<{ modelValue?: string }>()
const emit = defineEmits<{ select: [link: string] }>()

const loading = ref(false)
type GalleryItem = { ID: number; link: string; fid: string; originalName: string; relevanceId: number; filmName: string }

const list = ref<GalleryItem[]>([])
// 选 Logo / 海报时通常用手动上传的图片, 默认只列这一类
const source = ref('upload')
const caption = (img: GalleryItem) =>
    img.relevanceId > 0 ? (img.filmName || `视频 #${img.relevanceId}`) : (img.originalName || img.fid)
const page = reactive({ current: 1, pageSize: 24, total: 0 })

const load = async (current = 1) => {
    loading.value = true
    try {
        const resp = await http.get('/manage/file/list', { params: { current, source: source.value || undefined } })
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

// 每次打开时重新加载, 以包含刚上传的图片
watch(visible, (v) => { if (v) load(1) })

const pick = (link: string) => {
    emit('select', link)
    visible.value = false
}
</script>

<style scoped>
.gallery {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 12px;
  min-height: 120px;
}
.gallery-filter {
  margin-bottom: 12px;
}
.gallery-item {
  border: 2px solid transparent;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  background: var(--el-fill-color-light);
}
.gallery-item:hover {
  border-color: var(--el-color-primary-light-5);
}
.gallery-item.active {
  border-color: var(--el-color-primary);
}
.gallery-thumb {
  display: block;
  width: 100%;
  aspect-ratio: 1;
}
.gallery-caption {
  padding: 4px 6px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.gallery-failed {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gallery-empty {
  grid-column: 1 / -1;
}
.gallery-pagination {
  display: flex;
  justify-content: center;
  margin-top: 12px;
}
</style>
