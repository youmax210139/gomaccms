<template>
  <el-tabs v-model="active" class="lang-tabs">
    <el-tab-pane label="中文 (主)" name="">
      <slot/>
    </el-tab-pane>
    <el-tab-pane v-for="l in shown" :key="l.code" :label="l.name" :name="l.code">
      <p class="lang-tip" :style="{ marginLeft: labelWidth }">{{ tip }}</p>
      <el-form-item v-for="f in fields" :key="f.key" :label="f.label">
        <el-input v-if="f.textarea" v-model="i18n[l.code][f.key]" type="textarea" :autosize="{ minRows: f.rows ?? 2, maxRows: 12 }"
                  :maxlength="f.maxlength" :placeholder="placeholder(f)"/>
        <el-input v-else v-model="i18n[l.code][f.key]" :maxlength="f.maxlength" :placeholder="placeholder(f)"/>
      </el-form-item>
    </el-tab-pane>
  </el-tabs>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Lang } from '../utils/i18n'

// 一个可翻译的栏位: key 为译文中的栏位, src 为原文中对应的栏位 (默认同 key), 原文作为输入框的 placeholder
export type LangField = { key: string; label: string; src?: string; textarea?: boolean; rows?: number; maxlength?: number }

// 按语言编辑的页签: 「中文 (主)」页签放各页面自己的原文栏位 (默认插槽), 其余每个语言一个页签按 fields 生成
const props = withDefaults(defineProps<{
  languages: Lang[]
  i18n: Record<string, Record<string, any>>
  source: Record<string, any>
  fields: LangField[]
  labelWidth?: string
  // 翻译页签上方的提示; placeholders 为 false 时输入框不以原文作提示 (如 SEO 规则, 留空时不回退到中文)
  tip?: string
  placeholders?: boolean
}>(), { labelWidth: '90px', tip: '留空的字段使用中文 (主) 的内容', placeholders: true })

// 只显示已有译文对象的语言 (新增的语言由 watchI18nLangs 补上之前不渲染, 避免绑定到 undefined)
const shown = computed(() => props.languages.filter((l) => props.i18n[l.code]))
const placeholder = (f: LangField) => (props.placeholders ? String(props.source[f.src ?? f.key] ?? '') : '')
// 当前页签 ('' 为「中文 (主)」); 页面可用 v-model:active 控制 (如重新打开对话框时回到中文), 不绑定时由组件自己保存
const active = defineModel<string>('active', { default: '' })
</script>

<style scoped>
.lang-tabs {
  margin-bottom: 8px;
}
.lang-tip {
  margin-top: 0;
  margin-bottom: 10px;
  color: #909399;
  font-size: 12px;
}
</style>
