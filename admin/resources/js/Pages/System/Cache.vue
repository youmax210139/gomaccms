<template>
  <div>
    <p class="tip">页面即时渲染, 以下缓存只为加速; 内容新增 / 修改 / 删除时会自动清除相关缓存, 一般不需要手动刷新。</p>
    <div class="cache-list">
      <div v-for="c in caches" :key="c.target" class="cache-item">
        <div class="cache-text">
          <div class="cache-name">{{ c.label }}</div>
          <div class="cache-desc">{{ c.desc }}</div>
        </div>
        <el-button size="small" :type="c.target === 'all' ? 'danger' : undefined" plain :loading="busy === c.target" @click="refresh(c.target)">
          {{ c.target === 'all' ? '清除' : '刷新' }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import { http } from '../../utils/http'

defineOptions({ layout: AdminLayout })

const caches = [
  { target: 'home', label: '首页', desc: '各分类方案的首页数据' },
  { target: 'today', label: '今日内容', desc: '侧栏今日更新数与热搜词' },
  { target: 'rss', label: 'RSS', desc: '全部分类方案的 /rss.xml (单个方案可在「网站地图」中刷新)' },
  { target: 'search', label: '搜索结果', desc: '关键字搜索结果 (保留搜索次数)' },
  { target: 'all', label: '全部缓存', desc: '以上全部' },
]

const busy = ref('')
const refresh = async (target: string) => {
  busy.value = target
  try {
    const resp = await http.post(`/manage/cache/refresh/${target}`)
    ElMessage[resp.data.code === 0 ? 'success' : 'error']({ message: resp.data.msg })
  } finally {
    busy.value = ''
  }
}
</script>

<style scoped>
.tip {
  color: #909399;
  font-size: 13px;
  margin: 0 0 16px;
  line-height: 1.6;
}
.cache-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 12px;
}
.cache-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}
.cache-text {
  flex: 1;
}
.cache-name {
  font-weight: 600;
}
.cache-desc {
  margin-top: 2px;
  color: #909399;
  font-size: 12px;
}
</style>
