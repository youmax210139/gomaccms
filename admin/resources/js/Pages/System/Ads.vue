<template>
  <div>
    <el-tabs :model-value="String(scheme)" type="card" @tab-change="(name: string | number) => router.get('/manage/ad/list', { scheme: Number(name) })">
      <el-tab-pane v-for="s in schemes" :key="s.id" :name="String(s.id)" :label="s.name"/>
    </el-tabs>
    <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
    <p class="tip">每个分类方案各自一组图片广告 (支持 jpg / png / webp / gif 动图), 按广告位分组; 主题以
      <code>{maccms:ad slot="广告位"}...{/maccms:ad}</code> 显示, 默认主题的播放页使用 play_top 与 play_bottom。修改后前台立即生效。</p>
    <TableToolbar>
      <el-button type="primary" :icon="CirclePlus" @click="openForm(null)">添加广告</el-button>
    </TableToolbar>
    <el-table stripe :data="ads" style="width: 100%" size="default" table-layout="auto" empty-text="该方案还没有广告">
      <el-table-column label="广告位" min-width="150">
        <template #default="{ row }">
          <el-tag size="small" disable-transitions>{{ row.slot }}</el-tag>
          <div v-if="slots[row.slot]" class="sub">{{ slots[row.slot] }}</div>
        </template>
      </el-table-column>
      <el-table-column prop="name" label="名称" min-width="140"/>
      <el-table-column align="center" label="图片" min-width="200">
        <template #default="{ row }">
          <el-image class="ad-img" :src="row.image" :preview-src-list="[row.image]" preview-teleported fit="contain"/>
        </template>
      </el-table-column>
      <el-table-column label="链接" min-width="180" show-overflow-tooltip>
        <template #default="{ row }"><span v-if="row.link">{{ row.link }}</span><span v-else class="sub">不链接</span></template>
      </el-table-column>
      <el-table-column align="center" label="排序" width="80" prop="sort"/>
      <el-table-column align="center" label="状态" width="100">
        <template #default="{ row }">
          <el-switch :model-value="row.status" inline-prompt active-text="启用" inactive-text="禁用"
                     @change="(v: string | number | boolean) => changeStatus(row, Boolean(v))"/>
        </template>
      </el-table-column>
      <el-table-column align="center" label="操作" min-width="140">
        <template #default="{ row }">
          <el-button size="small" @click="openForm(row)">编辑</el-button>
          <el-button size="small" @click="delAd(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog v-model="dialog" width="640px" :title="form.id ? '修改广告' : '添加广告'" :confirm-text="form.id ? '保存' : '添加'" :loading="form.processing" @confirm="save">
      <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
      <el-form :model="form" label-width="80px">
        <el-form-item label="广告位" required>
          <el-select v-model="form.slot" filterable allow-create default-first-option placeholder="选择或输入, 如 play_top" style="width: 100%">
            <el-option v-for="(label, code) in slots" :key="code" :value="code" :label="`${code} · ${label}`"/>
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.name" maxlength="100" placeholder="后台辨识用, 也作为图片的替代文字"/>
        </el-form-item>
        <el-form-item label="图片" required>
          <div class="img-field">
            <el-image v-if="form.image" class="ad-img" :src="form.image" fit="contain"/>
            <el-input v-model="form.image" placeholder="图片地址 (站内 /upload/... 或 https://...)"/>
            <div class="img-actions">
              <el-upload :show-file-list="false" accept="image/jpeg,image/png,image/webp,image/gif" :http-request="upload">
                <el-button :loading="uploading">上传图片</el-button>
              </el-upload>
              <el-button @click="pickerVisible = true">从图库选择</el-button>
            </div>
          </div>
          <GalleryPicker v-model:visible="pickerVisible" :model-value="form.image" @select="(link: string) => (form.image = link)"/>
        </el-form-item>
        <el-form-item label="链接">
          <el-input v-model="form.link" maxlength="1024" placeholder="点击后打开的地址 (新窗口), 留空则不链接"/>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" :step="1" step-strictly/> <span class="field-tip">越小越靠前</span>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" inline-prompt active-text="启用" inactive-text="禁用"/>
        </el-form-item>
      </el-form>
    </AdminDialog>
  </div>
</template>

<script setup lang="ts">
import { CirclePlus } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'
import { router, useForm, usePage } from '@inertiajs/vue3'
import { ElMessage, type UploadRequestOptions } from 'element-plus'
import { http } from '../../utils/http'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import AdminDialog from '../../Components/AdminDialog.vue'
import TableToolbar from '../../Components/TableToolbar.vue'
import GalleryPicker from '../../Components/GalleryPicker.vue'
import { confirmAction } from '../../utils/confirm'

type Ad = { id: number; schemeId: number; slot: string; name: string; image: string; link: string; sort: number; status: boolean }

const props = defineProps<{
  scheme: number
  schemes: { id: number; name: string }[] | null
  ads: Ad[] | null
  slots: Record<string, string> | null
}>()
const page = usePage<{ errors?: { form?: string } }>()
defineOptions({ layout: AdminLayout })

const schemes = computed(() => props.schemes ?? [])
const ads = computed(() => props.ads ?? [])
const slots = computed(() => props.slots ?? {})

const empty = (): Ad => ({ id: 0, schemeId: props.scheme, slot: 'play_top', name: '', image: '', link: '', sort: 0, status: true })
const dialog = ref(false)
const form = useForm<Ad>(empty())
const openForm = (a: Ad | null) => {
  Object.assign(form, a ? { ...a } : empty())
  dialog.value = true
}
const save = () => {
  if (!form.slot.trim() || !form.name.trim() || !form.image.trim()) {
    ElMessage.warning('广告位、名称与图片不能为空')
    return
  }
  form.post('/manage/ad/save', {
    preserveScroll: true,
    onSuccess: () => { if (!page.props.errors?.form) dialog.value = false },
  })
}
const changeStatus = (a: Ad, status: boolean) => {
  router.post('/manage/ad/state', { id: a.id, status }, { preserveScroll: true })
}
const delAd = async (a: Ad) => {
  if (!await confirmAction(`确认删除广告「${a.name}」?`)) return
  router.get('/manage/ad/del', { id: a.id }, { preserveScroll: true })
}

const pickerVisible = ref(false)
const uploading = ref(false)
const upload = async (options: UploadRequestOptions) => {
  const data = new FormData()
  data.append('file', options.file)
  uploading.value = true
  try {
    const resp = await http.post('/manage/file/upload', data)
    if (resp.data.code === 0) form.image = resp.data.data
    else ElMessage.error(resp.data.msg || '上传失败')
  } finally {
    uploading.value = false
  }
}
</script>

<style scoped>
.tip {
  color: #909399;
  font-size: 13px;
  margin: 0 0 12px;
  line-height: 1.6;
}
.tip code {
  background: var(--el-fill-color-light);
  padding: 1px 6px;
  border-radius: 4px;
}
.sub {
  color: #909399;
  font-size: 12px;
  margin-top: 2px;
}
.ad-img {
  width: 180px;
  height: 60px;
  flex: none;
}
.img-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}
.img-actions {
  display: flex;
  gap: 8px;
}
.field-tip {
  margin-left: 8px;
  color: #909399;
  font-size: 12px;
}
</style>
