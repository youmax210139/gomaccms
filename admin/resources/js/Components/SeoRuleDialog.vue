<template>
  <AdminDialog v-model="visible" width="820px" :title="`SEO 规则 · ${schemeName}`" :loading="form.processing" @confirm="save">
    <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
    <div class="head">
      <p class="tip">页面的 SEO 依次取: 这里的规则 → 站点 SEO。每个栏位各自回退, 某语言留空时使用该语言的站点 SEO。</p>
      <el-button size="small" @click="fillPresets">填入预设值</el-button>
    </div>
    <div @focusin="onFocus">
      <el-tabs v-model="pageType">
        <el-tab-pane v-for="t in pageTypes" :key="t.key" :label="t.label" :name="t.key">
          <el-form label-width="110px">
            <LangTabs v-if="form.rules[t.key]" :languages="languages" :i18n="form.rules[t.key]" :source="form.rules[t.key]['zh-CN']"
                      :fields="fields" label-width="110px" :placeholders="false" tip="留空时使用该语言的站点 SEO">
              <el-form-item label="Title"><el-input v-model="form.rules[t.key]['zh-CN'].title" maxlength="255" :placeholder="titleHint(t.key)"/></el-form-item>
              <el-form-item label="Keywords"><el-input v-model="form.rules[t.key]['zh-CN'].keywords" maxlength="255"/></el-form-item>
              <el-form-item label="Description">
                <el-input v-model="form.rules[t.key]['zh-CN'].description" type="textarea" :autosize="{ minRows: 2, maxRows: 5 }" maxlength="500"/>
              </el-form-item>
            </LangTabs>
          </el-form>
          <div class="holders">
            <span class="holders-label">可用占位符 (点击插入到光标处):</span>
            <el-tag v-for="h in holdersOf(t.key)" :key="h" class="holder" size="small" effect="plain" disable-transitions
                    @mousedown.prevent @click="insert(`{${h}}`)">{{ '{' + h + '}' }}</el-tag>
          </div>
          <p v-if="titleHint(t.key)" class="hint">Title 没有设置时使用: {{ titleHint(t.key) }}</p>
        </el-tab-pane>
      </el-tabs>
    </div>
  </AdminDialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useForm, usePage } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import AdminDialog from './AdminDialog.vue'
import LangTabs, { type LangField } from './LangTabs.vue'
import { i18nOf, type Lang } from '../utils/i18n'

type RuleText = { title: string; keywords: string; description: string }
export type SeoRules = Record<string, Record<string, Partial<RuleText>>>

// 分类方案的 SEO 规则: 页面类型页签 × 语言页签 (中文 + 方案启用的翻译语言); 由页面通过 ref 调用 open
const props = defineProps<{ defaults: Record<string, string> | null }>()
const page = usePage<{ errors?: { form?: string } }>()

const pageTypes = [
  { key: 'home', label: '首页' }, { key: 'type', label: '分类' }, { key: 'show', label: '筛选' },
  { key: 'detail', label: '详情' }, { key: 'play', label: '播放' }, { key: 'search', label: '搜索' },
]
const fields: LangField[] = [
  { key: 'title', label: 'Title', maxlength: 255 },
  { key: 'keywords', label: 'Keywords', maxlength: 255 },
  { key: 'description', label: 'Description', textarea: true, maxlength: 500 },
]
const site = ['site_name', 'site_url', 'site_keywords', 'site_description', 'page', 'year']
const type = ['type_name']
const vod = ['vod_name', 'vod_sub', 'vod_year', 'vod_area', 'vod_lang', 'vod_class', 'vod_actor', 'vod_director', 'vod_remarks', 'vod_score', 'vod_content']
const holders: Record<string, string[]> = {
  home: site, search: site, type: [...site, ...type], show: [...site, ...type],
  detail: [...site, 'type_name', ...vod], play: [...site, 'type_name', ...vod, 'episode_name', 'episode_index'],
}
const holdersOf = (t: string) => holders[t] ?? site
const titleHint = (t: string) => props.defaults?.[t] ?? ''

const empty = (): RuleText => ({ title: '', keywords: '', description: '' })

// 预设值: 分类 / 筛选 / 详情 / 播放 (首页、搜索沿用站点 SEO). 标题的固定文字按语言, 其他语言只用占位符
const presetTitles: Record<string, Record<string, string>> = {
  'zh-CN': {
    type: '{type_name}大全 - 第{page}页 - {site_name}', show: '{type_name}筛选 - 第{page}页 - {site_name}',
    detail: '{vod_name}({vod_year}) 在线观看 - {site_name}', play: '{vod_name} {episode_name} 在线播放 - {site_name}',
  },
  vi: {
    type: 'Phim {type_name} - Trang {page} - {site_name}', show: 'Lọc phim {type_name} - Trang {page} - {site_name}',
    detail: '{vod_name} ({vod_year}) - Xem phim online - {site_name}', play: '{vod_name} {episode_name} - Xem phim - {site_name}',
  },
  en: {
    type: '{type_name} - Page {page} - {site_name}', show: 'Browse {type_name} - Page {page} - {site_name}',
    detail: '{vod_name} ({vod_year}) - Watch Online - {site_name}', play: '{vod_name} {episode_name} - Watch - {site_name}',
  },
  '': {
    type: '{type_name} - {page} - {site_name}', show: '{type_name} - {page} - {site_name}',
    detail: '{vod_name} ({vod_year}) - {site_name}', play: '{vod_name} {episode_name} - {site_name}',
  },
}
const vodKeywords = '{vod_name},{vod_sub},{vod_actor},{vod_director},{type_name}'
const presetOthers: Record<string, Omit<RuleText, 'title'>> = {
  type: { keywords: '{type_name},{site_keywords}', description: '{type_name} - {site_description}' },
  show: { keywords: '{type_name},{site_keywords}', description: '{type_name} - {site_description}' },
  detail: { keywords: vodKeywords, description: '{vod_name}: {vod_content}' },
  play: { keywords: vodKeywords, description: '{vod_name} {episode_name}: {vod_content}' },
}
// fillPresets 中文与方案启用的语言: 只填空着的栏位 (不覆盖已填的), 不自动保存
const fillPresets = () => {
  let n = 0
  const langs = ['zh-CN', ...languages.value.map((l) => l.code)]
  for (const [type, others] of Object.entries(presetOthers)) {
    for (const lang of langs) {
      const r = form.rules[type]?.[lang]
      if (!r) continue
      const preset: RuleText = { title: (presetTitles[lang] ?? presetTitles[''])[type], ...others }
      for (const k of ['title', 'keywords', 'description'] as const) {
        if (!r[k].trim()) {
          r[k] = preset[k]
          n++
        }
      }
    }
  }
  ElMessage[n ? 'success' : 'info'](n ? `已填入 ${n} 个栏位, 检查后保存` : '所有栏位都已有内容, 没有填入')
}
const visible = ref(false)
const pageType = ref('home')
const schemeName = ref('')
const languages = ref<Lang[]>([])
const form = useForm({ schemeId: 0, rules: {} as Record<string, Record<string, RuleText>> })

// open 每个页面类型: 中文 + 翻译语言补空白, 已存的 (包括方案暂时停用的语言) 保留
const open = (scheme: { id: number; name: string }, langs: Lang[], rules: SeoRules | null) => {
  const all = [{ code: 'zh-CN', name: '中文' }, ...langs]
  const r: Record<string, Record<string, RuleText>> = {}
  for (const t of pageTypes) r[t.key] = i18nOf(all, rules?.[t.key] as Record<string, RuleText> | undefined, empty)
  Object.assign(form, { schemeId: scheme.id, rules: r })
  schemeName.value = scheme.name
  languages.value = langs
  pageType.value = 'home'
  visible.value = true
}

// 占位符插入到最后获得焦点的输入框的光标处
let target: HTMLInputElement | HTMLTextAreaElement | null = null
const onFocus = (e: FocusEvent) => {
  const el = e.target
  if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) target = el
}
const insert = (text: string) => {
  if (!target || !target.isConnected) {
    ElMessage.info('请先点一下要插入的输入框')
    return
  }
  const el = target
  const start = el.selectionStart ?? el.value.length
  const end = el.selectionEnd ?? start
  el.value = el.value.slice(0, start) + text + el.value.slice(end)
  el.dispatchEvent(new Event('input')) // 同步到 v-model
  el.focus()
  el.setSelectionRange(start + text.length, start + text.length)
}

const save = () => {
  form.post('/manage/film/class/scheme/seo', {
    preserveScroll: true,
    onSuccess: () => { if (!page.props.errors?.form) visible.value = false },
  })
}

defineExpose({ open })
</script>

<style scoped>
.head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.tip {
  flex: 1;
  margin: 0 0 10px;
  color: #909399;
  font-size: 13px;
  line-height: 1.6;
}
.holders {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin: 4px 0 6px 110px;
}
.holders-label {
  color: #909399;
  font-size: 12px;
}
.holder {
  cursor: pointer;
}
.hint {
  margin: 0 0 0 110px;
  color: #909399;
  font-size: 12px;
}
</style>
