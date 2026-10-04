<template>
  <div>
    <el-tabs :model-value="String(scheme)" type="card" @tab-change="(name: string | number) => router.get('/manage/banner/list', { scheme: Number(name) })">
      <el-tab-pane v-for="s in schemes" :key="s.id" :name="String(s.id)" :label="s.name"/>
    </el-tabs>
    <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
    <p class="tip">每个分类方案各自一组首页海报, 前台只显示启用的海报, 显示几个由主题决定 (默认主题最多 6 个); 排序值小的在前。</p>
    <TableToolbar>
      <el-button type="primary" :icon="CirclePlus" @click="openForm(null)">添加海报</el-button>
    </TableToolbar>
    <el-table stripe :data="banners" style="width: 100%" size="default" table-layout="auto" empty-text="该方案还没有海报">
      <el-table-column label="海报名称" min-width="160">
        <template #default="{ row }">
          <div>{{ row.name }}</div>
          <div v-if="translated(row)" class="sub">{{ translated(row) }}</div>
        </template>
      </el-table-column>
      <el-table-column label="绑定视频" min-width="160">
        <template #default="{ row }">
          <template v-if="row.mid"><el-tag size="small" disable-transitions>ID {{ row.mid }}</el-tag> {{ vodNames[row.mid] ?? '(视频不存在)' }}</template>
          <span v-else class="sub">未绑定</span>
        </template>
      </el-table-column>
      <el-table-column align="center" label="视频海报">
        <template #default="{ row }">
          <el-image style="width: 180px; height: 80px" :src="row.poster" :preview-src-list="[row.poster]" preview-teleported fit="contain"/>
        </template>
      </el-table-column>
      <el-table-column align="center" label="视频封面">
        <template #default="{ row }">
          <el-image style="width: 60px; height: 80px" :src="row.picture" :preview-src-list="[row.picture]" preview-teleported fit="cover"/>
        </template>
      </el-table-column>
      <el-table-column align="center" label="排序" width="80">
        <template #default="{ row }">{{ row.sort }}</template>
      </el-table-column>
      <el-table-column align="center" label="状态" width="100">
        <template #default="{ row }">
          <el-switch :model-value="row.status" inline-prompt active-text="启用" inactive-text="禁用"
                     @change="(v: string | number | boolean) => changeStatus(row, Boolean(v))"/>
        </template>
      </el-table-column>
      <el-table-column align="center" label="操作" min-width="140">
        <template #default="{ row }">
          <el-button size="small" @click="openForm(row)">编辑</el-button>
          <el-button size="small" @click="delBanner(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog v-model="dialog" width="720px" :title="form.id ? '修改海报' : '添加海报'" :confirm-text="form.id ? '保存' : '添加'" :loading="form.processing" @confirm="save">
      <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
      <BannerForm :banner="form" :languages="languages" :scheme="scheme" :vod-name="formVodName" @upload="onUpload" @bind="(n: string) => (formVodName = n)"/>
    </AdminDialog>
  </div>
</template>

<script setup lang="ts">
import { CirclePlus } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'
import { router, useForm, usePage } from '@inertiajs/vue3'
import { http } from '../../utils/http'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import AdminDialog from '../../Components/AdminDialog.vue'
import { confirmAction } from '../../utils/confirm'
import TableToolbar from '../../Components/TableToolbar.vue'
import BannerForm, { type BannerData, type BannerText } from '../../Components/BannerForm.vue'
import { i18nOf, watchI18nLangs } from '../../utils/i18n'

type Banner = BannerData & { id: number; schemeId: number }

const props = defineProps<{
  scheme: number
  schemes: { id: number; name: string }[] | null
  banners: Banner[] | null
  languages: { code: string; name: string }[] | null
  vodNames: Record<string, string> | null
}>()
const page = usePage<{ errors?: { form?: string } }>()

defineOptions({ layout: AdminLayout })

const schemes = computed(() => props.schemes ?? [])
const banners = computed(() => props.banners ?? [])
const languages = computed(() => props.languages ?? [])

// 列表中显示已设置的译名
const translated = (b: Banner) => languages.value.map((l) => b.i18n?.[l.code]?.name).filter(Boolean).join(' / ')

const vodNames = computed(() => props.vodNames ?? {})
const emptyText = (): BannerText => ({ name: '' })
const empty = (): Banner => ({ id: 0, schemeId: props.scheme, mid: 0, name: '', poster: '', picture: '', sort: 0, status: true, i18n: i18nOf(languages.value, null, emptyText) })

const dialog = ref(false)
const formVodName = ref('')
const form = useForm<Banner>(empty())
watchI18nLangs(languages, () => form.i18n, emptyText)
const openForm = (b: Banner | null) => {
  const data = b ? { ...b, i18n: i18nOf(languages.value, b.i18n, emptyText) } : empty()
  formVodName.value = b?.mid ? (vodNames.value[b.mid] ?? '') : ''
  Object.assign(form, data)
  dialog.value = true
}
const save = () => {
  form.post('/manage/banner/save', {
    preserveScroll: true,
    onSuccess: () => { if (!page.props.errors?.form) dialog.value = false },
  })
}
const changeStatus = (b: Banner, status: boolean) => {
  router.post('/manage/banner/state', { id: b.id, status }, { preserveScroll: true })
}
const delBanner = async (b: Banner) => {
  if (!await confirmAction(`确认删除海报「${b.name}」?`)) return
  router.get('/manage/banner/del', { id: b.id })
}

const onUpload = async (file: File, field: 'poster' | 'picture') => {
  const formData = new FormData()
  formData.append('file', file)
  const resp = await http.post('/manage/file/upload', formData)
  if (resp.data.code === 0) {
    form[field] = resp.data.data
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
.sub {
  color: #909399;
  font-size: 12px;
  margin-top: 2px;
}
</style>
