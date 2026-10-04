<template>
  <el-result icon="warning" title="没有权限" sub-title="当前账号没有访问该页面的权限, 请联系创始管理员授权">
    <template #extra>
      <el-button type="primary" @click="router.visit('/manage/index')">返回首页</el-button>
    </template>
  </el-result>
</template>

<script setup lang="ts">
import { router } from '@inertiajs/vue3'
import AdminLayout from '../Layouts/AdminLayout.vue'

defineOptions({ layout: AdminLayout })
</script>
