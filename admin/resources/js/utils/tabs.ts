import { reactive } from 'vue'
import { HOME, pathOf, titleOf } from './menu'

// 页签栏状态: 最多同时打开 MAX_TABS 个页签 (含固定的首页), 超出时关闭最久未访问的页签.
// 存在 sessionStorage 中, 刷新页面后保留, 关闭浏览器标签页后清空.

export const MAX_TABS = 10
const STORAGE_KEY = 'gomaccms-admin-tabs'

export type Tab = {
    path: string   // 页签标识 (不含查询参数)
    url: string    // 最近一次访问的完整地址 (含分页/筛选参数), 切回页签时恢复
    title: string
    visitedAt: number
}

const homeTab = (): Tab => ({ path: HOME.path, url: HOME.path, title: HOME.title, visitedAt: 0 })

const load = (): Tab[] => {
    try {
        const saved = JSON.parse(sessionStorage.getItem(STORAGE_KEY) ?? '[]') as Tab[]
        if (Array.isArray(saved) && saved.length > 0) return saved
    } catch {
        // 读取失败 (隐私模式等) 时从空白页签开始
    }
    return [homeTab()]
}

const save = () => {
    try {
        sessionStorage.setItem(STORAGE_KEY, JSON.stringify(state.tabs))
    } catch {
        // 忽略
    }
}

export const state = reactive({ tabs: load() })

export const isPinned = (tab: Tab) => tab.path === HOME.path

// openTab 记录一次页面访问: 已有页签则更新地址, 否则新增 (菜单外的页面不产生页签)
export const openTab = (url: string) => {
    const path = pathOf(url)
    const title = titleOf(path)
    if (!title) return
    const now = Date.now()
    const existing = state.tabs.find((t) => t.path === path)
    if (existing) {
        existing.url = url
        existing.visitedAt = now
    } else {
        state.tabs.push({ path, url, title, visitedAt: now })
        while (state.tabs.length > MAX_TABS) {
            const oldest = state.tabs
                .filter((t) => !isPinned(t) && t.path !== path)
                .sort((a, b) => a.visitedAt - b.visitedAt)[0]
            if (!oldest) break
            state.tabs.splice(state.tabs.indexOf(oldest), 1)
        }
    }
    save()
}

// closeTab 关闭页签, 返回关闭后应切换到的页签 (关闭的是当前页签时); 首页不可关闭
export const closeTab = (path: string, activePath: string): Tab | undefined => {
    const i = state.tabs.findIndex((t) => t.path === path)
    if (i < 0 || isPinned(state.tabs[i])) return
    state.tabs.splice(i, 1)
    save()
    if (path !== activePath) return
    return state.tabs[i] ?? state.tabs[i - 1]
}

// closeOtherTabs 只保留首页与指定页签
export const closeOtherTabs = (path: string) => {
    state.tabs = state.tabs.filter((t) => isPinned(t) || t.path === path)
    save()
}

// resetTabs 退出登录 / 进入登录页时清空页签, 下一个登录的账号从只有首页开始
export const resetTabs = () => {
    state.tabs = [homeTab()]
    try {
        sessionStorage.removeItem(STORAGE_KEY)
    } catch {
        // 忽略
    }
}
