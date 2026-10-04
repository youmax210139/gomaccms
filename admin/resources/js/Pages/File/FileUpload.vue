<template>
  <div>
    <p v-if="pageProps.props.errors?.form" class="form-error">{{ pageProps.props.errors.form }}</p>
    <TableToolbar>
      <el-upload :show-file-list="false" accept="image/*" multiple :http-request="customUpload">
        <el-button type="primary" :icon="Upload" :loading="uploading > 0">上传图片</el-button>
      </el-upload>
      <template #right>
        <el-radio-group :model-value="source" @change="(v) => reload(1, String(v))">
          <el-radio-button value="">全部</el-radio-button>
          <el-radio-button value="upload">手动上传</el-radio-button>
          <el-radio-button value="poster">视频封面</el-radio-button>
        </el-radio-group>
      </template>
    </TableToolbar>

    <div class="gallery">
      <div v-for="(img, i) in list" :key="img.ID" class="gallery-card">
        <el-image :src="img.link" fit="cover" class="gallery-thumb" :preview-src-list="links" :initial-index="i" preview-teleported>
          <template #error><div class="gallery-failed">文件不存在</div></template>
        </el-image>
        <div class="gallery-meta">
          <div class="gallery-name" :title="displayName(img)">{{ displayName(img) }}</div>
          <div class="gallery-sub">
            <el-tag size="small" :type="img.relevanceId > 0 ? 'warning' : 'success'" disable-transitions>
              {{ img.relevanceId > 0 ? '视频封面' : '手动上传' }}
            </el-tag>
            <span class="gallery-time">{{ img.createdAt }}</span>
          </div>
        </div>
        <el-button class="gallery-del" type="danger" :icon="Delete" circle size="small" @click="delImage(img)"/>
      </div>
      <el-empty v-if="list.length === 0" description="暂无图片" class="gallery-empty"/>
    </div>

    <div class="pagination">
      <el-pagination background layout="total, prev, pager, next" :total="page.total" :page-size="page.pageSize"
                     :current-page="page.current" hide-on-single-page @current-change="(pg: number) => reload(pg)"/>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Delete, Upload } from '@element-plus/icons-vue'
import { ElMessage, type UploadRequestOptions } from 'element-plus'
import { router, usePage } from '@inertiajs/vue3'
import { http } from '../../utils/http'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import TableToolbar from '../../Components/TableToolbar.vue'
import { confirmAction } from '../../utils/confirm'

type GalleryItem = {
  ID: number
  link: string
  fid: string
  originalName: string
  relevanceId: number
  filmName: string
  createdAt: string
}

const props = defineProps<{
  list: GalleryItem[] | null
  page: { pageSize: number; current: number; pageCount: number; total: number }
  source: string
}>()
const pageProps = usePage<{ errors?: { form?: string } }>()

defineOptions({ layout: AdminLayout })

// preserveState 导航不会重新挂载组件, 列表一律从 props 派生
const list = computed(() => props.list ?? [])
const links = computed(() => list.value.map((f) => f.link))

// 视频封面显示视频名称; 手动上传显示原始档名, 旧图片没有记录原始档名时显示文件标识
const displayName = (img: GalleryItem) => {
    if (img.relevanceId > 0) return img.filmName || `视频 #${img.relevanceId}`
    return img.originalName || img.fid
}

const reload = (current: number, source = props.source) => {
    router.get('/manage/file/upload', { current, source: source || undefined }, { preserveState: true, preserveScroll: true })
}

const uploading = ref(0)
const customUpload = async (options: UploadRequestOptions) => {
    const formData = new FormData()
    formData.append('file', options.file)
    uploading.value++
    try {
        const resp = await http.post('/manage/file/upload', formData)
        if (resp.data.code === 0) {
            ElMessage.success({ message: resp.data.msg })
            reload(1)
        } else {
            ElMessage.error({ message: resp.data.msg })
        }
    } finally {
        uploading.value--
    }
}

const delImage = async (img: GalleryItem) => {
    const msg = img.relevanceId > 0
        ? `这是视频「${img.filmName || img.relevanceId}」的封面, 删除后该视频将改用采集站的原始图片地址。确认删除?`
        : `确认删除图片「${displayName(img)}」? 如果它正被用作 Logo 或海报, 页面上将无法显示。`
    if (!await confirmAction(msg)) return
    router.get('/manage/file/del', { id: img.ID }, { preserveScroll: true })
}
</script>

<style scoped>
.gallery {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
}
.gallery-card {
  position: relative;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  overflow: hidden;
  background: var(--el-bg-color);
}
.gallery-thumb {
  display: block;
  width: 100%;
  aspect-ratio: 4 / 3;
  background: var(--el-fill-color-light);
}
.gallery-failed {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.gallery-meta {
  padding: 8px 10px;
}
.gallery-name {
  font-size: 13px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.gallery-sub {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
}
.gallery-time {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gallery-del {
  position: absolute;
  top: 8px;
  right: 8px;
  opacity: 0;
  transition: opacity 0.15s;
}
.gallery-card:hover .gallery-del {
  opacity: 1;
}
/* 触控设备没有 hover, 删除按钮常驻 */
@media (hover: none) {
  .gallery-del {
    opacity: 1;
  }
}
.gallery-empty {
  grid-column: 1 / -1;
}
.pagination {
  display: flex;
  justify-content: center;
  padding: 20px 0;
}
</style>
