import { watch, type WatchSource } from 'vue'

// 后台编辑译文的语言 (方案启用的、原文以外的语言)
export type Lang = { code: string; name: string }

// i18nOf 编辑用的译文: 当前语言补上空白, 已存的译文 (包括方案暂时停用的语言) 全部保留, 缺少的栏位补空
export function i18nOf<T extends object>(languages: Lang[], stored: Record<string, Partial<T>> | null | undefined, empty: () => T): Record<string, T> {
  const out: Record<string, T> = {}
  for (const l of languages) out[l.code] = empty()
  for (const [code, t] of Object.entries(stored ?? {})) out[code] = { ...empty(), ...t }
  return out
}

// watchI18nLangs 同一页面中方案新增了语言时 (保存方案不会重新挂载页面), 为新语言补上空白译文
export function watchI18nLangs<T extends object>(languages: WatchSource<Lang[]>, i18n: () => Record<string, T>, empty: () => T) {
  watch(languages, (langs) => {
    const m = i18n()
    for (const l of langs) if (!m[l.code]) m[l.code] = empty()
  })
}
