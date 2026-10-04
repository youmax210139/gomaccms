<template>
  <div>
    <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
    <el-tabs v-model="activeTab">
      <el-tab-pane label="基本设置" name="basic">
        <el-form :model="form" label-width="140px" class="basic-form">
          <el-form-item label="采集间隔">
            <el-input-number v-model="form.collectInterval" :min="0" :max="3600" controls-position="right"/>
            <span class="hint inline">单位秒, 建议设置为 3 秒以上; 0 为不限制。采集接口单独设置了间隔时以接口为准</span>
          </el-form-item>
          <el-form-item label="后台每页数">
            <el-input-number v-model="form.pageSize" :min="5" :max="200" controls-position="right"/>
            <span class="hint inline">影视信息、图库管理等列表每页显示的条数</span>
          </el-form-item>
          <el-form-item label="入库重复规则">
            <el-checkbox-group v-model="form.duplicateRule">
              <el-checkbox v-for="(label, key) in rules" :key="key" :value="key" :disabled="key === 'name'">{{ label }}</el-checkbox>
            </el-checkbox-group>
            <div class="hint">不同采集接口的视频, 所选栏位都相同时视为同一部, 合并播放地址; 年份、豆瓣ID 为空时不比对</div>
          </el-form-item>
          <el-form-item label="后台登录验证码">
            <el-radio-group v-model="form.loginCaptcha">
              <el-radio :value="false">关闭</el-radio>
              <el-radio :value="true">开启</el-radio>
            </el-radio-group>
          </el-form-item>
          <LangTabs :languages="allLangs" :i18n="form.hintI18n" :source="form" :fields="hintFields" label-width="140px">
            <el-form-item label="关闭提示" required>
              <el-input v-model="form.hint" type="textarea" :autosize="{ minRows: 2, maxRows: 5 }" maxlength="500" show-word-limit
                        placeholder="网站关闭时前台显示的提示"/>
              <div class="hint">所有域名共用, 按访客的语言显示; 在「方案管理」中关闭某个网站时显示</div>
            </el-form-item>
          </LangTabs>
          <el-form-item>
            <el-button type="primary" :loading="form.processing" @click="update">保存</el-button>
            <el-button @click="form.reset()">还原</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="方案管理" name="domain">
        <p class="tip">一个分类方案可以有多个域名: 同一方案的域名共用分类、海报与语系, 各自有独立的 Theme、站点信息与网站状态;
          未配置的域名使用默认方案的「未配置的域名」行。方案启用一种语言时为单语系 (前台不显示语言切换), 启用多种时前台可切换。</p>
        <TableToolbar v-if="can('category.scheme')">
          <el-button type="primary" :icon="CirclePlus" @click="schemeDialogs?.openNew()">新增方案</el-button>
        </TableToolbar>
        <div v-for="s in schemes" :key="s.id" class="scheme-block">
          <div class="scheme-head">
            <div class="scheme-title">
              <span class="scheme-name">{{ s.name }}</span>
              <el-tag v-if="s.id == 1" size="small" type="info" disable-transitions>默认</el-tag>
              <span class="scheme-meta">语言: {{ schemeLangs(s.id) }} · 分类 {{ schemeCounts[s.id] ?? 0 }} 个</span>
            </div>
            <div class="scheme-actions">
              <template v-if="can('category.scheme')">
                <el-button size="small" @click="schemeDialogs?.openLangs(s)">语系</el-button>
                <el-button size="small" @click="seoRuleDialog?.open(s, langsOfScheme(s.id), props.seoRules?.[s.id] ?? null)">SEO</el-button>
                <el-button size="small" @click="schemeDialogs?.openInfo(s)">修改</el-button>
                <el-tooltip v-if="s.id != 1" :disabled="!rowsOf(s.id).length" content="请先把使用此方案的域名移到其他方案" placement="top">
                  <span class="btn-wrap"><el-button size="small" :disabled="rowsOf(s.id).length > 0" @click="schemeDialogs?.remove(s)">删除</el-button></span>
                </el-tooltip>
              </template>
              <el-button size="small" type="primary" plain :icon="CirclePlus" @click="openAdd(s.id)">添加域名</el-button>
            </div>
          </div>
          <el-table stripe :data="rowsOf(s.id)" size="default" table-layout="auto" empty-text="还没有域名使用此方案">
            <el-table-column label="域名" min-width="160">
              <template #default="{ row }">
                <template v-if="row.isDefault">
                  <el-tag type="info" disable-transitions>默认</el-tag>
                  <span class="default-desc">未配置的域名</span>
                </template>
                <template v-else>{{ row.domain }}</template>
              </template>
            </el-table-column>
            <el-table-column align="center" label="Theme">
              <template #default="{ row }">
                <el-tag :type="themes.includes(row.theme) ? 'primary' : 'danger'" disable-transitions>
                  {{ row.theme }}{{ themes.includes(row.theme) ? '' : ' (不可用)' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="siteName" label="网站名称"/>
            <el-table-column prop="seoTitle" label="SEO Title" show-overflow-tooltip/>
            <el-table-column align="center" label="网站状态" width="100">
              <template #default="{ row }">
                <el-switch :model-value="row.state" inline-prompt active-text="开启" inactive-text="关闭"
                           @change="(v: string | number | boolean) => changeState(row, Boolean(v))"/>
              </template>
            </el-table-column>
            <el-table-column align="center" label="操作" width="210">
              <template #default="{ row }">
                <el-button size="small" @click="openSeo(row)">SEO</el-button>
                <el-button size="small" @click="row.isDefault ? openDefault() : openEdit(row)">编辑</el-button>
                <el-button v-if="!row.isDefault" size="small" @click="delDomain(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>
    </el-tabs>

    <SchemeDialogs ref="schemeDialogs" :languages="props.languages"/>
    <SeoRuleDialog ref="seoRuleDialog" :defaults="props.seoDefaults"/>

    <AdminDialog v-model="defaultDialog" width="760px" title="默认站点 (未配置的域名)" :loading="defaultForm.processing" @confirm="saveDefault">
      <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
      <el-form :model="defaultForm" label-width="140px">
        <el-form-item label="Theme">
          <el-tag type="info" disable-transitions>default</el-tag>
        </el-form-item>
        <el-form-item label="分类方案">
          <el-tag type="info" disable-transitions>{{ schemeName(1) }}</el-tag>
        </el-form-item>
        <el-form-item label="网站状态">
          <el-switch v-model="defaultForm.state" inline-prompt active-text="开启" inactive-text="关闭"/>
        </el-form-item>
        <SiteInfoFields :info="defaultForm">
          <template #info>
            <el-form-item label="网站备注">
              <el-input v-model="defaultForm.remark" type="textarea" :autosize="{ minRows: 2, maxRows: 5 }" maxlength="500" show-word-limit placeholder="仅后台可见的一般备注"/>
            </el-form-item>
          </template>
        </SiteInfoFields>
      </el-form>
    </AdminDialog>

    <AdminDialog v-model="domainDialog" width="760px" :title="domainForm.id ? '修改域名' : '添加域名'" :loading="domainForm.processing" @confirm="saveDomain">
      <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
      <el-form :model="domainForm" label-width="140px">
        <el-form-item label="域名" required>
          <el-input v-model="domainForm.domain" placeholder="movie.example.com (不含 http:// 与端口)"/>
        </el-form-item>
        <el-form-item label="Theme" required>
          <el-select v-model="domainForm.theme" placeholder="选择主题">
            <el-option v-for="t in themes" :key="t" :label="t" :value="t"/>
          </el-select>
        </el-form-item>
        <el-form-item label="分类方案" required>
          <el-select v-model="domainForm.schemeId">
            <el-option v-for="s in schemes" :key="s.id" :label="s.name" :value="s.id"/>
          </el-select>
          <span class="hint inline">前台导航与视频按此方案的分类展示; 此方案的语言: {{ schemeLangs(domainForm.schemeId) }}, 用方案的「语系」按钮调整</span>
        </el-form-item>
        <el-form-item label="网站状态">
          <el-switch v-model="domainForm.state" inline-prompt active-text="开启" inactive-text="关闭"/>
        </el-form-item>
        <el-form-item v-if="!domainForm.id" label="网站名称" required>
          <el-input v-model="domainForm.siteName" maxlength="50" show-word-limit/>
          <div class="hint">SEO、法律信息与各语言的文字, 添加后用该域名的「SEO」按钮编辑</div>
        </el-form-item>
        <SiteInfoFields :info="domainForm"/>
      </el-form>
    </AdminDialog>

    <AdminDialog v-model="seoDialog" width="760px" :title="`SEO · ${seoName}`" :loading="seoForm.processing" @confirm="saveSeo">
      <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
      <p class="tip">分类方案: {{ schemeName(seoScheme) }} · 语言: {{ schemeLangs(seoScheme) }}; 翻译页签中留空的字段, 前台使用中文 (主) 的内容。</p>
      <el-form :model="seoForm" label-width="140px">
        <el-form-item v-if="seoForm.id">
          <el-button size="small" @click="copyDefaultText">从「默认」复制</el-button>
        </el-form-item>
        <SiteTextFields :key="seoKey" :info="seoForm" :languages="langsOfScheme(seoScheme)"/>
      </el-form>
    </AdminDialog>
  </div>
</template>

<script setup lang="ts">
import { CirclePlus } from '@element-plus/icons-vue'
import { computed, ref, watch } from 'vue'
import { router, useForm, usePage } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import AdminDialog from '../../Components/AdminDialog.vue'
import { confirmAction } from '../../utils/confirm'
import TableToolbar from '../../Components/TableToolbar.vue'
import SiteInfoFields, { emptySiteText, type SiteInfo } from '../../Components/SiteInfoFields.vue'
import SiteTextFields from '../../Components/SiteTextFields.vue'
import LangTabs, { type LangField } from '../../Components/LangTabs.vue'
import { i18nOf, watchI18nLangs, type Lang } from '../../utils/i18n'
import SchemeDialogs from '../../Components/SchemeDialogs.vue'
import SeoRuleDialog, { type SeoRules } from '../../Components/SeoRuleDialog.vue'
import { useCan } from '../../utils/menu'

// basic: 全局配置 = 「默认」站点 (SiteInfo / remark / state) + 关闭提示 + 后台设置
type Basic = SiteInfo & {
  remark: string
  state: boolean
  hint: string
  collectInterval: number
  pageSize: number
  loginCaptcha: boolean
  duplicateRule: string[] | null
  hintI18n: Record<string, string> | null
}
type Domain = SiteInfo & { ID: number; domain: string; theme: string; state: boolean; schemeId: number }
type Row = SiteInfo & { isDefault: boolean; ID: number; domain: string; theme: string; state: boolean; schemeId: number }

const props = defineProps<{
  basic: Basic
  domains: Domain[] | null
  themes: string[] | null
  schemes: { id: number; name: string; sort: number; defaultLang: string; langs: string[] | null; domains?: string[] | null }[] | null
  languages: { code: string; name: string; enabled: boolean }[] | null
  schemeCounts: Record<string, number> | null
  seoRules: Record<string, SeoRules> | null
  seoDefaults: Record<string, string> | null
  rules: Record<string, string> | null
}>()
const page = usePage<{ errors?: { form?: string } }>()
const can = useCan()

defineOptions({ layout: AdminLayout })

const activeTab = ref('basic')

// ---- 语言: 「系统语言」中启用的、原文以外的语言; 方案的翻译语言为其中该方案启用的
const allLangs = computed<Lang[]>(() => (props.languages ?? []).filter((l) => l.enabled && l.code !== 'zh-CN'))
const langsOfScheme = (id: number): Lang[] => {
    const codes = (props.schemes ?? []).find((s) => s.id == (id || 1))?.langs ?? []
    return allLangs.value.filter((l) => codes.includes(l.code))
}

// langs: 编辑时要补上空白译文的语言 (列表显示不需要)
const pickSiteInfo = (s: Partial<SiteInfo>, langs: Lang[] = []): SiteInfo => ({
    siteName: s.siteName ?? '', logo: s.logo ?? '', seoTitle: s.seoTitle ?? '',
    keyword: s.keyword ?? '', describe: s.describe ?? '', serviceEmail: s.serviceEmail ?? '',
    analyticsCode: s.analyticsCode ?? '', legalInfo: s.legalInfo ?? '',
    i18n: i18nOf(langs, s.i18n, emptySiteText),
})

// ---- 基本设置 (关闭提示的译文在表单中为 {lang: {hint}}, 与 LangTabs 一致, 提交时转回 {lang: hint})
type HintText = { hint: string }
const emptyHint = (): HintText => ({ hint: '' })
const hintFields: LangField[] = [{ key: 'hint', label: '关闭提示', textarea: true, maxlength: 500 }]
const pickSettings = (b: Basic) => ({
    hint: b.hint ?? '',
    hintI18n: i18nOf(allLangs.value, Object.fromEntries(Object.entries(b.hintI18n ?? {}).map(([k, v]) => [k, { hint: v }])), emptyHint),
    collectInterval: b.collectInterval ?? 0,
    pageSize: b.pageSize || 20,
    loginCaptcha: b.loginCaptcha ?? false,
    duplicateRule: b.duplicateRule?.length ? [...b.duplicateRule] : ['name', 'year'],
})
const form = useForm(pickSettings(props.basic))
// POST redirect 回本页时组件不会重新挂载 (preserveState), 需同步最新的 props, 「还原」回到已保存的值
watch(() => props.basic, (b) => {
    const v = pickSettings(b)
    form.defaults(v)
    Object.assign(form, v)
})
watchI18nLangs(allLangs, () => form.hintI18n, emptyHint)
const update = () => {
    form.transform((data) => ({ ...data, hintI18n: Object.fromEntries(Object.entries(data.hintI18n).map(([k, v]) => [k, v.hint])) }))
        .post('/manage/config/basic/update', { preserveScroll: true })
}

// ---- 网域与 Theme: 第一行为「默认」站点
const domains = computed(() => props.domains ?? [])
const themes = computed(() => props.themes ?? [])
const schemes = computed(() => props.schemes ?? [])
const rules = computed(() => props.rules ?? {})
const schemeName = (id: number) => schemes.value.find((s) => s.id == (id || 1))?.name ?? `#${id}`
const langName = (code: string) => (props.languages ?? []).find((l) => l.code === code)?.name ?? code
// 方案启用的语言与默认语言, 如「中文、Tiếng Việt (默认: 中文)」
const schemeLangs = (id: number) => {
  const s = schemes.value.find((x) => x.id == (id || 1))
  if (!s) return ''
  const def = s.defaultLang || 'zh-CN'
  return `${(s.langs?.length ? s.langs : ['zh-CN']).map(langName).join('、')} (默认: ${langName(def)})`
}
const schemeCounts = computed(() => props.schemeCounts ?? {})
const schemeDialogs = ref<InstanceType<typeof SchemeDialogs> | null>(null)
const seoRuleDialog = ref<InstanceType<typeof SeoRuleDialog> | null>(null)
// 各方案区块中的域名 (按方案ID分组); 方案已不存在的域名 (前台按默认方案处理) 归入默认方案
const rowsByScheme = computed(() => {
    const m: Record<number, Row[]> = {}
    for (const s of schemes.value) m[s.id] = []
    for (const r of rows.value) (m[r.schemeId] ?? m[1] ?? (m[1] = [])).push(r)
    return m
})
const rowsOf = (id: number) => rowsByScheme.value[id] ?? []
const rows = computed<Row[]>(() => [
    { isDefault: true, ID: 0, domain: '', theme: 'default', state: props.basic.state, schemeId: 1, ...pickSiteInfo(props.basic) },
    ...domains.value.map((d) => ({
        isDefault: false, ID: d.ID, domain: d.domain, theme: d.theme, state: d.state, schemeId: d.schemeId || 1, ...pickSiteInfo(d),
    })),
])

const changeState = async (row: Row, state: boolean) => {
    if (!state && !await confirmAction(`确认关闭「${row.isDefault ? '默认站点' : row.domain}」? 关闭后前台页面将显示关闭提示。`)) return
    router.post('/manage/config/domain/state', { id: row.ID, state }, { preserveScroll: true })
}

// 默认站点
const defaultDialog = ref(false)
const defaultForm = useForm({ ...pickSiteInfo({}), remark: '', state: true })
const openDefault = () => {
    Object.assign(defaultForm, pickSiteInfo(props.basic, langsOfScheme(1)), { remark: props.basic.remark ?? '', state: props.basic.state })
    defaultDialog.value = true
}
const saveDefault = () => {
    if (!defaultForm.siteName.trim()) {
        ElMessage.warning('网站名称不能为空')
        return
    }
    defaultForm.post('/manage/config/default/update', {
        preserveScroll: true,
        onSuccess: () => {
            if (!page.props.errors?.form) defaultDialog.value = false
        },
    })
}

// 域名
const emptyDomain = () => ({ id: 0, domain: '', theme: '', state: true, schemeId: 1, ...pickSiteInfo({}) })
const domainDialog = ref(false)
const domainForm = useForm(emptyDomain())

const openAdd = (schemeId: number) => {
    Object.assign(domainForm, emptyDomain(), {
        schemeId, theme: themes.value.includes('default') ? 'default' : (themes.value[0] ?? ''),
    })
    domainDialog.value = true
}
const openEdit = (d: Row) => {
    Object.assign(domainForm, { id: d.ID, domain: d.domain, theme: d.theme, state: d.state, schemeId: d.schemeId || 1 }, pickSiteInfo(d, langsOfScheme(d.schemeId)))
    domainDialog.value = true
}
const saveDomain = () => {
    if (!domainForm.domain.trim()) {
        ElMessage.warning('域名不能为空')
        return
    }
    if (!domainForm.theme) {
        ElMessage.warning('请选择主题')
        return
    }
    if (!domainForm.id && !domainForm.siteName.trim()) {
        ElMessage.warning('网站名称不能为空')
        return
    }
    const url = domainForm.id ? '/manage/config/domain/update' : '/manage/config/domain/add'
    domainForm.post(url, {
        preserveScroll: true,
        onSuccess: () => {
            if (!page.props.errors?.form) domainDialog.value = false
        },
    })
}
const delDomain = async (d: Row) => {
    if (!await confirmAction(`确认删除域名「${d.domain}」?`)) return
    router.get('/manage/config/domain/del', { id: d.ID })
}

// ---- SEO 弹窗: 网站名称、SEO、法律信息及各语言译文 (「默认」站点为 ID 0, 使用默认方案的语言)
const seoDialog = ref(false)
const seoScheme = ref(1)
const seoName = ref('')
const seoKey = ref(0)
const textOf = (s: Partial<SiteInfo>, langs: Lang[]) => ({
    siteName: s.siteName ?? '', seoTitle: s.seoTitle ?? '', keyword: s.keyword ?? '', describe: s.describe ?? '',
    legalInfo: s.legalInfo ?? '', i18n: i18nOf(langs, s.i18n, emptySiteText),
})
const seoForm = useForm({ id: 0, ...textOf({}, []) })
const openSeo = (row: Row) => {
    seoScheme.value = row.isDefault ? 1 : (row.schemeId || 1)
    seoName.value = row.isDefault ? '默认站点 (未配置的域名)' : row.domain
    Object.assign(seoForm, { id: row.ID }, textOf(row, langsOfScheme(seoScheme.value)))
    seoKey.value++ // 重新挂载页签, 每次打开都从「中文 (主)」开始
    seoDialog.value = true
}
const copyDefaultText = () => {
    Object.assign(seoForm, textOf(props.basic, langsOfScheme(seoScheme.value)))
}
const saveSeo = () => {
    if (!seoForm.siteName.trim()) {
        ElMessage.warning('网站名称不能为空')
        return
    }
    seoForm.post('/manage/config/seo/update', {
        preserveScroll: true,
        onSuccess: () => {
            if (!page.props.errors?.form) seoDialog.value = false
        },
    })
}
</script>

<style scoped>
.scheme-block {
  margin-bottom: 20px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
}
.scheme-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 14px;
  background: var(--el-fill-color-light);
}
.scheme-title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.scheme-name {
  font-weight: 600;
  font-size: 15px;
}
.scheme-meta {
  color: #909399;
  font-size: 12px;
}
.scheme-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.scheme-actions .el-button + .el-button {
  margin-left: 0;
}
.btn-wrap {
  display: inline-flex;
}
.tip {
  color: #909399;
  font-size: 13px;
  margin: 0 0 16px;
}
.basic-form {
  max-width: 900px;
}
.hint {
  width: 100%;
  color: #909399;
  font-size: 12px;
  line-height: 1.6;
}
.hint.inline {
  width: auto;
  margin-left: 12px;
}
.default-desc {
  margin-left: 8px;
  color: #909399;
  font-size: 13px;
}
</style>
