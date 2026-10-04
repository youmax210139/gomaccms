<template>
  <LangTabs :languages="languages" :i18n="info.i18n" :source="info" :fields="siteFields" :label-width="labelWidth">
    <el-form-item label="网站名称" required>
      <el-input v-model="info.siteName" maxlength="50" show-word-limit/>
    </el-form-item>
    <el-form-item label="SEO Title">
      <el-input v-model="info.seoTitle" maxlength="100" show-word-limit placeholder="留空时使用网站名称"/>
    </el-form-item>
    <el-form-item label="SEO Keywords">
      <el-input v-model="info.keyword" maxlength="200" show-word-limit placeholder="多个关键字以逗号分隔"/>
    </el-form-item>
    <el-form-item label="SEO Description">
      <el-input v-model="info.describe" type="textarea" :autosize="{ minRows: 2, maxRows: 5 }" maxlength="500" show-word-limit/>
    </el-form-item>
    <el-form-item label="法律信息">
      <el-input v-model="info.legalInfo" type="textarea" :autosize="{ minRows: 2, maxRows: 6 }" maxlength="2000" show-word-limit placeholder="显示在前台页面底部"/>
    </el-form-item>
  </LangTabs>
</template>

<script setup lang="ts">
import LangTabs, { type LangField } from './LangTabs.vue'
import type { Lang } from '../utils/i18n'
import type { SiteText } from './SiteInfoFields.vue'

// SEO 弹窗: 网站名称、SEO、法律信息, 「中文 (主)」+ 站点所用方案的每个翻译语言一个页签 (留空的字段前台使用原文)
withDefaults(defineProps<{
  info: SiteText & { i18n: Record<string, SiteText> }
  languages: Lang[]
  labelWidth?: string
}>(), { labelWidth: '140px' })

const siteFields: LangField[] = [
  { key: 'siteName', label: '网站名称', maxlength: 50 },
  { key: 'seoTitle', label: 'SEO Title', maxlength: 100 },
  { key: 'keyword', label: 'SEO Keywords', maxlength: 200 },
  { key: 'describe', label: 'SEO Description', textarea: true, maxlength: 500 },
  { key: 'legalInfo', label: '法律信息', textarea: true, maxlength: 2000 },
]
</script>
