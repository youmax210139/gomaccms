<template>
  <!-- 采集进度弹窗: 关闭只是收起到右下角页签, 不会中止采集 -->
  <el-dialog :model-value="!!task" width="760px" align-center append-to-body class="admin-dialog"
             :title="task ? `采集 · ${task.sourceName}` : ''" @update:model-value="(v: boolean) => { if (!v) collectTasks.openId = '' }">
    <template v-if="task">
      <div class="task-head">
        <el-tag :type="stateOf(task).type" disable-transitions>{{ stateOf(task).label }}</el-tag>
        <span>{{ task.mode }}</span>
        <el-tag v-if="task.trigger && task.trigger !== '手动'" size="small" type="info" disable-transitions>{{ task.trigger }}</el-tag>
        <span class="task-elapsed">{{ elapsed(task) }}</span>
      </div>
      <div class="task-line">
        <template v-if="task.typeCount">
          当前分类 <b>{{ task.typeName || task.typeId }}</b> ({{ task.typeIndex }}/{{ task.typeCount }})
          · 第 <b>{{ task.page }}</b>/{{ task.pageCount }} 页<template v-if="task.total"> · 共 <b>{{ task.total }}</b> 条</template>
        </template>
        <template v-else>准备中…</template>
      </div>
      <el-progress :percentage="percent(task)" :status="task.state === 'done' ? 'success' : task.state === 'failed' ? 'exception' : undefined"/>
      <div class="task-line">
        新增 <b class="c-add">{{ task.added }}</b> · 更新 <b class="c-update">{{ task.updated }}</b> · 跳过 <b>{{ task.skipped }}</b>
        <template v-if="task.failedPages"> · 失败 <b class="c-fail">{{ task.failedPages }}</b> 页 (可在采集记录中重试)</template>
      </div>
      <p v-if="task.state !== 'running'" class="task-message">{{ task.message }}</p>
      <div ref="logBox" class="task-log">
        <div v-for="(line, i) in task.logs ?? []" :key="i" :class="lineClass(line)">{{ line }}</div>
      </div>
    </template>
    <template #footer>
      <el-button v-if="canResume(task)" type="primary" @click="resumeCollect(task!.sourceId)">从中断处继续</el-button>
      <el-button v-if="task?.state === 'running'" type="danger" :loading="task.stopping" @click="confirmStop(task.sourceId)">
        {{ task.stopping ? '正在中止…' : '中止采集' }}
      </el-button>
      <el-button @click="collectTasks.openId = ''">{{ task?.state === 'running' ? '收起 (后台继续采集)' : '关闭' }}</el-button>
    </template>
  </el-dialog>

  <!-- 右下角的采集任务页签 -->
  <div v-if="tabs.length" class="task-tabs">
    <div v-for="t in tabs" :key="t.sourceId" class="task-tab" :class="t.state" @click="openCollectTask(t.sourceId)">
      <el-icon v-if="t.state === 'running'" class="is-loading"><Loading/></el-icon>
      <span class="task-tab-name">{{ t.sourceName }}</span>
      <span class="task-tab-text">{{ tabText(t) }}</span>
      <el-icon v-if="t.state !== 'running'" class="task-tab-close" title="关闭" @click.stop="dismissCollectTask(t.sourceId)"><Close/></el-icon>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { Close, Loading } from '@element-plus/icons-vue'
import { confirmAction } from '../utils/confirm'
import {
  canResume, collectTasks, dismissCollectTask, initCollectTasks, openCollectTask, resumeCollect, stopCollect, type CollectProgress,
} from '../utils/collectTasks'

onMounted(initCollectTasks)

const task = computed(() => (collectTasks.openId ? collectTasks.tasks[collectTasks.openId] : undefined))
const tabs = computed(() => collectTasks.watched
  .filter((id) => id !== collectTasks.openId && collectTasks.tasks[id])
  .map((id) => collectTasks.tasks[id]))

const stateOf = (p: CollectProgress) => ({
  running: { label: p.stopping ? '正在中止' : '采集中', type: 'primary' as const },
  done: { label: '已完成', type: 'success' as const },
  stopped: { label: '已中止', type: 'warning' as const },
  interrupted: { label: '已中断', type: 'warning' as const },
  failed: { label: '失败', type: 'danger' as const },
}[p.state])
const percent = (p: CollectProgress) => {
  if (p.state === 'done') return 100
  if (!p.typeCount) return 0
  // 按分类与分类内页数估算整体进度
  const inType = p.pageCount > 0 ? p.page / p.pageCount : 0
  return Math.min(100, Math.round(((p.typeIndex - 1 + inType) / p.typeCount) * 100))
}
const elapsed = (p: CollectProgress) => {
  const s = Math.max(0, (p.finishedAt || Math.floor(Date.now() / 1000)) - p.startedAt)
  return s >= 60 ? `已用 ${Math.floor(s / 60)} 分 ${s % 60} 秒` : `已用 ${s} 秒`
}
const tabText = (p: CollectProgress) => (p.state === 'running'
  ? `${p.page}/${p.pageCount} 页 · ${percent(p)}%`
  : stateOf(p).label)
const lineClass = (line: string) => (line.includes('新增') ? 'c-add' : line.includes('更新') ? 'c-update'
  : line.includes('失败') ? 'c-fail' : line.includes('跳过') ? 'c-skip' : '')

const confirmStop = async (id: string) => {
  if (!await confirmAction('确认中止本次采集? 当前页采集完后停止, 已入库的视频会保留。')) return
  stopCollect(id)
}

// 新日志到达时滚动到底部
const logBox = ref<HTMLElement>()
watch(() => [collectTasks.openId, task.value?.logs?.length], () => nextTick(() => {
  if (logBox.value) logBox.value.scrollTop = logBox.value.scrollHeight
}))
</script>

<style scoped>
.task-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.task-elapsed {
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.task-line {
  margin: 8px 0;
  font-size: 13px;
}
.task-message {
  margin: 8px 0;
  font-size: 13px;
  color: var(--el-text-color-regular);
}
.task-log {
  height: 360px;
  overflow-y: auto;
  padding: 8px 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  line-height: 1.7;
}
.c-add {
  color: var(--el-color-success);
}
.c-update {
  color: var(--el-color-primary);
}
.c-fail {
  color: var(--el-color-danger);
}
.c-skip {
  color: var(--el-text-color-secondary);
}
.task-tabs {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-index: 2000;
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: flex-end;
}
.task-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: 320px;
  padding: 8px 12px;
  border-radius: 6px;
  background: var(--el-bg-color);
  border-left: 4px solid var(--el-color-primary);
  box-shadow: var(--el-box-shadow-light);
  font-size: 13px;
  cursor: pointer;
}
.task-tab.done {
  border-left-color: var(--el-color-success);
}
.task-tab.stopped,
.task-tab.interrupted {
  border-left-color: var(--el-color-warning);
}
.task-tab.failed {
  border-left-color: var(--el-color-danger);
}
.task-tab-name {
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.task-tab-text {
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.task-tab-close {
  color: var(--el-text-color-secondary);
}
.task-tab-close:hover {
  color: var(--el-color-danger);
}
</style>
