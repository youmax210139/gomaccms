import { reactive } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'
import { http } from './http'

// 采集任务 (参照苹果 CMS 的采集窗口): 点击采集后弹窗显示进度, 关闭弹窗后在右下角留下小页签,
// 点击页签重新打开弹窗, 可在弹窗中中止采集. 进度由后端保存在内存中, 这里每 2 秒轮询一次.
export type CollectProgress = {
  sourceId: string; sourceName: string; mode: string; trigger: string; hours: number
  state: 'running' | 'done' | 'failed' | 'stopped' | 'interrupted'; message: string; resumeTypeId: number; resumePage: number
  startedAt: number; finishedAt: number; typeIndex: number; typeCount: number; typeId: number; typeName: string
  page: number; pageCount: number; total: number; added: number; updated: number; skipped: number; failedPages: number
  failedList: { typeId: number; page: number }[] | null
  logs: string[] | null; stopping: boolean
}

export const collectTasks = reactive({
  tasks: {} as Record<string, CollectProgress>,
  // 本次打开后台以来关注的任务 (自己开始的, 或发现正在进行的), 显示为右下角页签, 关闭页签后移除
  watched: [] as string[],
  openId: '',
  starting: {} as Record<string, boolean>,
})

let timer: ReturnType<typeof setTimeout> | undefined
let loaded = false

// 关闭过的页签 (采集接口ID:开始时间), 存在 localStorage, 重新整理后不再出现
const DISMISSED_KEY = 'gomaccms.collect.dismissed'
const dismissed = new Set<string>((() => {
  try { return JSON.parse(localStorage.getItem(DISMISSED_KEY) ?? '[]') as string[] } catch { return [] }
})())
const taskKey = (p: CollectProgress) => `${p.sourceId}:${p.startedAt}`

const watch = (id: string) => {
  if (!collectTasks.watched.includes(id)) collectTasks.watched.push(id)
}

// 采集结束时 (进度弹窗没开着) 在右上角通知
const notifyFinished = (p: CollectProgress) => {
  if (collectTasks.openId === p.sourceId) return
  const type = p.state === 'done' ? 'success' : p.state === 'stopped' ? 'warning' : 'error'
  ElNotification({ title: `${p.sourceName} · ${p.mode}`, message: p.message, type, duration: 8000 })
}

export const refreshCollectTasks = async () => {
  clearTimeout(timer)
  try {
    const resp = await http.get('/manage/collect/progress')
    if (resp.data.code === 0) {
      const m: Record<string, CollectProgress> = {}
      for (const p of resp.data.data ?? []) {
        m[p.sourceId] = p
        const prev = collectTasks.tasks[p.sourceId]
        if (prev?.state === 'running' && p.state !== 'running' && collectTasks.watched.includes(p.sourceId)) notifyFinished(p)
        // 进行中的采集, 以及服务重启后被中断、尚未关闭页签的采集, 都显示为页签
        if (p.state === 'running' || (p.state === 'interrupted' && !dismissed.has(taskKey(p)))) watch(p.sourceId)
      }
      collectTasks.tasks = m
    }
  } catch { /* 下次再试 */ }
  if (Object.values(collectTasks.tasks).some((p) => p.state === 'running')) {
    timer = setTimeout(refreshCollectTasks, 2000)
  }
}

// 后台版面挂载时调用一次, 找回正在进行的采集任务 (切换页面或重新整理后)
export const initCollectTasks = () => {
  if (loaded) return
  loaded = true
  refreshCollectTasks()
}

// 采集接口是否正在采集 (含已点击、尚未收到回应的), 用于禁用采集按钮防止重复点击
export const isCollecting = (id: string) => !!collectTasks.starting[id] || collectTasks.tasks[id]?.state === 'running'

export const startCollect = async (id: string, time: number) => {
  if (isCollecting(id)) {
    collectTasks.openId = id
    return
  }
  collectTasks.starting[id] = true
  try {
    const resp = await http.post('/manage/spider/start', { id, time })
    if (resp.data.code !== 0) {
      ElMessage.error({ message: resp.data.msg })
      return
    }
    watch(id)
    collectTasks.openId = id
    await refreshCollectTasks()
  } finally {
    delete collectTasks.starting[id]
  }
}

// 可以从中断处继续的采集 (已中止 / 已中断 / 失败, 且记录了中断位置)
export const canResume = (p?: CollectProgress) => !!p && ['stopped', 'interrupted', 'failed'].includes(p.state) && p.resumeTypeId > 0

export const resumeCollect = async (id: string) => {
  if (isCollecting(id)) return
  collectTasks.starting[id] = true
  try {
    const resp = await http.post('/manage/spider/resume', { id })
    if (resp.data.code !== 0) {
      ElMessage.error({ message: resp.data.msg })
      return
    }
    watch(id)
    collectTasks.openId = id
    await refreshCollectTasks()
  } finally {
    delete collectTasks.starting[id]
  }
}

// 重采一笔采集记录中失败的页 (开始后同样显示进度弹窗)
export const retryFailedPages = async (sourceId: string, logId: number) => {
  if (isCollecting(sourceId)) return false
  collectTasks.starting[sourceId] = true
  try {
    const resp = await http.post('/manage/spider/retry', { id: logId })
    if (resp.data.code !== 0) {
      ElMessage.error({ message: resp.data.msg })
      return false
    }
    watch(sourceId)
    collectTasks.openId = sourceId
    await refreshCollectTasks()
    return true
  } finally {
    delete collectTasks.starting[sourceId]
  }
}

export const stopCollect = async (id: string) => {
  const resp = await http.post('/manage/spider/stop', { id })
  ElMessage[resp.data.code === 0 ? 'success' : 'error']({ message: resp.data.msg })
  refreshCollectTasks()
}

export const openCollectTask = (id: string) => {
  collectTasks.openId = id
}

export const dismissCollectTask = (id: string) => {
  collectTasks.watched = collectTasks.watched.filter((w) => w !== id)
  const p = collectTasks.tasks[id]
  if (p) {
    dismissed.add(taskKey(p))
    try { localStorage.setItem(DISMISSED_KEY, JSON.stringify([...dismissed].slice(-100))) } catch { /* 忽略 */ }
  }
}

// 采集结果状态的名称与标签颜色 (进度弹窗、采集记录、后台首页共用)
export const collectStates: Record<string, { label: string; type: 'primary' | 'success' | 'warning' | 'danger' | 'info' }> = {
  running: { label: '采集中', type: 'primary' },
  done: { label: '已完成', type: 'success' },
  stopped: { label: '已中止', type: 'warning' },
  interrupted: { label: '已中断', type: 'warning' },
  failed: { label: '失败', type: 'danger' },
}
