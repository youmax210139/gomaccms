<template>
  <!-- 数据表格上方的操作栏: 左侧放新增/批量等 CRUD 操作, 右侧 (#right) 放次要操作 -->
  <div class="table-toolbar">
    <div class="table-toolbar-left">
      <slot />
    </div>
    <div v-if="$slots.right" class="table-toolbar-right">
      <slot name="right" />
    </div>
  </div>
</template>

<style scoped>
.table-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}
.table-toolbar-left,
.table-toolbar-right {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
/* el-button 相邻时自带 margin-left, 由 gap 统一间距 */
.table-toolbar :deep(.el-button + .el-button) {
  margin-left: 0;
}
</style>
