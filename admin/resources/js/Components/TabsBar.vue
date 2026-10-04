<template>
  <div class="tabs-bar">
    <el-scrollbar ref="scrollbar" class="tabs-scroll">
      <div class="tabs-strip">
        <div
          v-for="tab in state.tabs"
          :key="tab.path"
          :data-path="tab.path"
          class="tab"
          :class="{ active: tab.path === activePath }"
          @click="visit(tab)"
        >
          <span>{{ tab.title }}</span>
          <el-icon v-if="!isPinned(tab)" class="tab-close" @click.stop="close(tab)"><Close /></el-icon>
        </div>
      </div>
    </el-scrollbar>
    <el-dropdown trigger="click" @command="onCommand">
      <el-button text>操作<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item command="refresh" :icon="Refresh">刷新当前页</el-dropdown-item>
          <el-dropdown-item command="others" :icon="Close">关闭其他页签</el-dropdown-item>
          <el-dropdown-item command="all" :icon="CircleClose">关闭全部页签</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { router, usePage } from '@inertiajs/vue3'
import { ArrowDown, CircleClose, Close, Refresh } from '@element-plus/icons-vue'
import { HOME, pathOf } from '../utils/menu'
import { closeOtherTabs, closeTab, isPinned, openTab, state, type Tab } from '../utils/tabs'

const page = usePage()
const scrollbar = ref<{ wrapRef?: HTMLElement }>()

const activePath = computed(() => pathOf(page.url))

// 每次页面访问 (含分页/筛选等查询参数变化) 都更新页签, 并把当前页签滚动到可见区域
watch(() => page.url, (url) => {
    openTab(url)
    nextTick(() => {
        scrollbar.value?.wrapRef?.querySelector(`[data-path="${activePath.value}"]`)?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
    })
}, { immediate: true })

const visit = (tab: Tab) => {
    if (tab.path !== activePath.value) router.visit(tab.url)
}

const close = (tab: Tab) => {
    const next = closeTab(tab.path, activePath.value)
    if (next) router.visit(next.url)
}

const onCommand = (command: string) => {
    if (command === 'refresh') {
        router.reload()
    } else if (command === 'others') {
        closeOtherTabs(activePath.value)
    } else if (command === 'all') {
        closeOtherTabs(HOME.path)
        if (activePath.value !== HOME.path) router.visit(HOME.path)
    }
}
</script>

<style scoped>
.tabs-bar {
  display: flex;
  align-items: center;
  gap: 8px;
}
.tabs-scroll {
  flex: 1;
  min-width: 0;
}
.tabs-strip {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 0;
}
.tab {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  height: 32px;
  padding: 0 12px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  font-size: 14px;
  color: var(--el-text-color-regular);
  background: #ffffff;
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
}
.tab:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary-light-5);
}
.tab.active {
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
  color: #ffffff;
}
.tab-close {
  border-radius: 50%;
  font-size: 12px;
}
.tab-close:hover {
  background: rgba(0, 0, 0, 0.15);
}
</style>
