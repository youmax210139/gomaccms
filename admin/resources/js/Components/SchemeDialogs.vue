<template>
  <AdminDialog v-model="dialog" width="520px" :title="title" :loading="form.processing" @confirm="save">
    <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
    <el-form :model="form" label-width="80px">
      <template v-if="mode !== 'langs'">
        <el-form-item label="名称" required><el-input v-model="form.name" maxlength="60" placeholder="如 动漫站方案"/></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" :step="1" step-strictly/> <span class="field-tip">越小越靠前</span></el-form-item>
      </template>
      <template v-if="mode !== 'info'">
        <el-form-item label="启用语言" required>
          <p v-if="droppedLangs.length" class="form-error">语言 {{ droppedLangs.map(langName).join('、') }} 已在「系统语言」中停用, 保存后将从此方案移除</p>
          <el-checkbox-group v-model="form.langs">
            <el-checkbox v-for="l in enabledLanguages" :key="l.code" :value="l.code">{{ l.name }}</el-checkbox>
          </el-checkbox-group>
          <div class="field-note">只列出「系统语言」中启用的语言; 启用一种时前台不显示语言切换</div>
        </el-form-item>
        <el-form-item label="默认语言" required>
          <el-select v-model="form.defaultLang" style="width: 200px">
            <el-option v-for="l in enabledLanguages.filter((x) => form.langs.includes(x.code))" :key="l.code" :label="l.name" :value="l.code"/>
          </el-select>
          <span class="field-tip">访客没有选过语言时使用</span>
        </el-form-item>
        <el-form-item v-if="mode === 'langs'" label="生效域名">
          <span class="field-note">{{ domains.length ? domains.join('、') : '暂无域名使用此方案' }}</span>
        </el-form-item>
      </template>
    </el-form>
  </AdminDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { router, useForm, usePage } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import AdminDialog from './AdminDialog.vue'
import { confirmAction } from '../utils/confirm'

export type Scheme = { id: number; name: string; sort: number; defaultLang: string; langs: string[] | null; domains?: string[] | null }
type Language = { code: string; name: string; enabled: boolean }

// 站群管理「方案管理」页签的方案对话框: 新增方案、修改 (名称 / 排序)、语系 (启用语言 / 默认语言)、删除;
// 由页面通过 ref 调用 openNew / openInfo / openLangs / remove
const props = defineProps<{ languages: Language[] | null }>()
const page = usePage<{ errors?: { form?: string } }>()

const languages = computed(() => props.languages ?? [])
const enabledLanguages = computed(() => languages.value.filter((l) => l.enabled))
const langName = (code: string) => languages.value.find((l) => l.code === code)?.name ?? code

type Mode = 'new' | 'info' | 'langs'
const mode = ref<Mode>('new')
const dialog = ref(false)
const domains = ref<string[]>([])
const title = computed(() => ({ new: '新增分类方案', info: '修改分类方案', langs: `语系 · ${form.name}` })[mode.value])
const form = useForm({ id: 0, name: '', sort: 0, defaultLang: 'zh-CN', langs: ['zh-CN'] as string[] })
// 方案中已停用的语言 (打开对话框时从表单中移除, 保存后生效)
const droppedLangs = ref<string[]>([])

const open = (m: Mode, s: Scheme | null) => {
  const enabled = enabledLanguages.value.map((l) => l.code)
  const saved = s?.langs?.length ? s.langs : ['zh-CN']
  droppedLangs.value = saved.filter((c) => !enabled.includes(c))
  const langs = saved.filter((c) => enabled.includes(c))
  if (!langs.length) langs.push('zh-CN')
  const def = s?.defaultLang && langs.includes(s.defaultLang) ? s.defaultLang : langs[0]
  Object.assign(form, { id: s?.id ?? 0, name: s?.name ?? '', sort: s?.sort ?? 0, defaultLang: def, langs })
  domains.value = s?.domains ?? []
  mode.value = m
  dialog.value = true
}
// 默认语言必须是启用的语言之一
watch(() => form.langs, (langs) => {
  if (langs.length && !langs.includes(form.defaultLang)) form.defaultLang = langs[0]
})

const save = () => {
  if (!form.name.trim()) {
    ElMessage.warning('方案名称不能为空')
    return
  }
  if (!form.langs.length) {
    ElMessage.warning('至少启用一种语言')
    return
  }
  form.post('/manage/film/class/scheme/save', {
    preserveScroll: true,
    onSuccess: () => { if (!page.props.errors?.form) dialog.value = false },
  })
}

const remove = async (s: Scheme) => {
  if (!await confirmAction(`确认删除分类方案「${s.name}」? 方案中的全部分类以及视频与这些分类的关联、指向它们的采集绑定都会一并删除 (视频本身不删除)。`)) return
  router.get('/manage/film/class/scheme/del', { id: s.id }, { preserveState: true, preserveScroll: true })
}

defineExpose({
  openNew: () => open('new', null),
  openInfo: (s: Scheme) => open('info', s),
  openLangs: (s: Scheme) => open('langs', s),
  remove,
})
</script>

<style scoped>
.field-tip {
  margin-left: 8px;
  color: #909399;
  font-size: 12px;
}
.field-note {
  width: 100%;
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
}
</style>
