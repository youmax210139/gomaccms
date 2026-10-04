<template>
  <!-- 后台统一弹窗: 水平 + 垂直居中, 内容过高时弹窗内滚动; 默认底部为「取消 / 确认」按钮 -->
  <el-dialog v-model="visible" :title="title" :width="width" align-center append-to-body class="admin-dialog">
    <slot />
    <template #footer>
      <slot name="footer">
        <div class="admin-dialog-footer">
          <div><slot name="footer-extra" /></div>
          <div>
            <el-button @click="visible = false">取消</el-button>
            <el-button type="primary" :loading="loading" @click="$emit('confirm')">{{ confirmText }}</el-button>
          </div>
        </div>
      </slot>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  title: string
  width?: string
  confirmText?: string
  loading?: boolean
}>(), {
  width: '600px',
  confirmText: '保存',
  loading: false,
})

defineEmits<{ confirm: [] }>()

const visible = defineModel<boolean>({ required: true })
</script>

<style>
/* el-dialog 会被传送到 body, 这里不能用 scoped 样式 */
.admin-dialog {
  max-width: calc(100vw - 32px);
}
.admin-dialog .el-dialog__body {
  max-height: calc(100vh - 200px);
  overflow-y: auto;
}
.admin-dialog-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
