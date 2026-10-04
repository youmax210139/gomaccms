<template>
  <div class="header-actions">
    <el-dropdown trigger="click" placement="bottom-end">
      <div class="user">
        <!-- 未设置头像时显示通用的使用者图标 -->
        <el-avatar v-if="avatarUrl" :size="30" :src="avatarUrl" alt="avatar" />
        <el-avatar v-else :size="30" :icon="UserFilled" class="default-avatar" />
        <span class="user-name">{{ currentUser.nickName }}</span>
        <el-icon><ArrowDown /></el-icon>
      </div>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item :icon="User" @click="router.visit('/manage/user/profile')">个人资料</el-dropdown-item>
          <el-dropdown-item :icon="SwitchButton" divided @click="logout">退出登录</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </div>
</template>

<script setup lang="ts">
import { ArrowDown, SwitchButton, User, UserFilled } from '@element-plus/icons-vue'
import { computed } from 'vue'
import { router } from '@inertiajs/vue3'
import { http } from '../utils/http'
import { resetTabs } from '../utils/tabs'

const props = defineProps<{
  currentUser: { nickName: string; avatar: string }
}>()

const avatarUrl = computed(() => {
    const a = props.currentUser.avatar
    return a && a !== 'empty' ? a : ''
})

const logout = async () => {
    await http.get('/logout')
    resetTabs()
    router.visit('/login')
}
</script>

<style scoped>
.header-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.user {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: 8px;
  padding: 3px 8px 3px 3px;
  border-radius: 20px;
  cursor: pointer;
}
.user:hover {
  background: var(--el-fill-color-light);
}
.user-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
}
.default-avatar {
  background: var(--el-color-info-light-5);
  color: #fff;
}
</style>
