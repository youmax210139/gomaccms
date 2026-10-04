<template>
  <div class="admin-layout">
    <aside class="admin-sidebar" :class="{ collapsed }">
      <Sidebar :site="page.props.site" :collapse="collapsed" :user="page.props.currentUser" />
    </aside>
    <div class="admin-main">
      <header class="admin-topbar">
        <button class="topbar-btn" :title="collapsed ? '展开菜单' : '收起菜单'" @click="collapsed = !collapsed">
          <el-icon :size="18"><Expand v-if="collapsed" /><Fold v-else /></el-icon>
        </button>
        <TabsBar class="admin-tabs" />
        <ManageHeader :current-user="page.props.currentUser" />
      </header>
      <main class="admin-content">
        <div class="admin-container">
          <slot />
        </div>
      </main>
    </div>
    <CollectTasks />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { usePage } from '@inertiajs/vue3'
import { Expand, Fold } from '@element-plus/icons-vue'
import Sidebar from '../Components/Sidebar.vue'
import ManageHeader from '../Components/ManageHeader.vue'
import TabsBar from '../Components/TabsBar.vue'
import CollectTasks from '../Components/CollectTasks.vue'

const page = usePage<{
  site: { siteName: string; logo: string }
  currentUser: { nickName: string; avatar: string; founder: boolean; permissions: string[] | null }
}>()

const collapsed = ref(false)
</script>

<style scoped>
.admin-layout {
  display: flex;
  height: 100vh;
  background: #f5f7fa;
}
.admin-sidebar {
  flex-shrink: 0;
  width: 220px;
  background: #ffffff;
  border-right: 1px solid var(--el-border-color-light);
  transition: width 0.2s;
}
.admin-sidebar.collapsed {
  width: 64px;
}
.admin-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.admin-topbar {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 56px;
  padding: 0 16px 0 8px;
  background: #ffffff;
  border-bottom: 1px solid var(--el-border-color-light);
}
.topbar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.topbar-btn:hover {
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
}
.admin-tabs {
  flex: 1;
  min-width: 0;
}
.admin-content {
  flex: 1;
  overflow: auto;
  padding: 16px;
}
/* 所有后台页面共用的内容卡片 (白底 + 圆角 + 内边距), 页面本身不再各自设置 */
.admin-container {
  box-sizing: border-box;
  min-height: 100%;
  padding: 20px;
  background: #ffffff;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}
</style>
