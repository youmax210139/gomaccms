<template>
  <div class="sidebar">
    <a class="sidebar-brand" href="/manage/index" @click.prevent="inertiaRouter.visit('/manage/index')">
      <img v-if="site?.logo" class="brand-logo" :src="site.logo" alt="logo" />
      <span v-show="!collapse" class="brand-name">控制台</span>
    </a>
    <el-scrollbar class="sidebar-menu">
      <el-menu :default-active="activePath" :default-openeds="openeds" :collapse="collapse" :collapse-transition="false" @select="onSelect">
        <el-menu-item :index="HOME.path">
          <el-icon><House /></el-icon>
          <template #title>{{ HOME.title }}</template>
        </el-menu-item>
        <el-sub-menu v-for="group in groups" :key="group.key" :index="group.key">
          <template #title>
            <el-icon><component :is="group.icon" /></el-icon>
            <span>{{ group.title }}</span>
          </template>
          <el-menu-item v-for="item in group.items" :key="item.path" :index="item.path">{{ item.title }}</el-menu-item>
        </el-sub-menu>
      </el-menu>
    </el-scrollbar>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { router as inertiaRouter, usePage } from '@inertiajs/vue3'
import { House } from '@element-plus/icons-vue'
import { HOME, pathOf, visibleGroups, type Access } from '../utils/menu'

const props = defineProps<{
  site: { siteName: string; logo: string }
  collapse: boolean
  user?: Access
}>()

const page = usePage()
const groups = computed(() => visibleGroups(props.user))
const activePath = computed(() => pathOf(page.url))
// 默认展开当前页面所在的分组
const openeds = computed(() => groups.value.filter((g) => g.items.some((i) => i.path === activePath.value)).map((g) => g.key))

const onSelect = (index: string) => {
    inertiaRouter.visit(index)
}
</script>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.sidebar-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  height: 56px;
  padding: 0 16px;
  color: var(--el-text-color-primary);
  text-decoration: none;
}
.brand-logo {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  object-fit: cover;
}
.brand-name {
  font-size: 20px;
  font-weight: 600;
  white-space: nowrap;
}
.sidebar-menu {
  flex: 1;
}
.sidebar-menu :deep(.el-menu) {
  border-right: none;
}
</style>
