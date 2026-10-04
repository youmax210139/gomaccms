<template>
  <div>
    <div class="browse-head">
      <el-button link :icon="Back" @click="router.get('/manage/collect/list')">返回列表</el-button>
      <span class="source-name">【{{ source.name }}】{{ source.uri }}{{ source.params }}</span>
    </div>
    <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>

    <div class="class-grid">
      <div class="class-item">
        <a :class="{ active: !query.t }" @click="visit({ t: 0, pg: 1 })">查看全部资源</a>
      </div>
      <div v-for="c in classes" :key="c.typeId" class="class-item">
        <a :class="{ active: query.t == c.typeId }" @click="visit({ t: c.typeId, pg: 1 })">{{ c.typeName }}</a>
        <a :class="['bind', { bound: c.categoryIds?.length, inherited: c.inherited }]" :title="(c.categories ?? []).join('\n')" @click="openBind(c)">[{{ bindLabel(c) }}]</a>
      </div>
    </div>

    <div class="search-bar">
      <el-input v-model="keyword" placeholder="关键字" clearable style="width: 220px" @keyup.enter="visit({ wd: keyword, pg: 1 })"/>
      <el-button type="primary" @click="visit({ wd: keyword, pg: 1 })">查询</el-button>
    </div>

    <el-table stripe :data="list" row-key="vod_id" size="default" table-layout="auto" empty-text="暂无资源" @selection-change="(rows: any[]) => selected = rows">
      <el-table-column type="selection" width="48"/>
      <el-table-column prop="vod_name" label="名称"/>
      <el-table-column prop="type_name" label="分类" width="120"/>
      <el-table-column prop="vod_play_from" label="来源" width="140" show-overflow-tooltip/>
      <el-table-column label="时间" width="180">
        <template #default="scope"><span class="time">{{ scope.row.vod_time }}</span></template>
      </el-table-column>
    </el-table>

    <div class="browse-foot">
      <div v-if="can('collect.run')">
        <el-button :icon="Plus" :disabled="selected.length === 0" @click="collectSelected">采选中</el-button>
        <el-button :icon="Plus" :disabled="isCollecting(source.id)" @click="collectTime(24, '采当天')">采当天</el-button>
        <el-button :icon="Plus" :disabled="isCollecting(source.id)" @click="collectTime(-1, '采全部')">采全部</el-button>
      </div>
      <div v-else/>
      <el-pagination layout="total, prev, pager, next, jumper" background :total="paging.total"
                     :page-count="paging.pageCount" :current-page="paging.page" @current-change="(pg: number) => visit({ pg })"/>
    </div>

    <AdminDialog v-model="bindV" width="520px" :title="`绑定分类: ${bindForm.typeName}`" confirm-text="保存" @confirm="saveBind">
      <el-select v-model="bindForm.categoryIds" placeholder="选择分类 (可多选, 可跨分类方案)" multiple filterable clearable style="width: 100%">
        <el-option-group v-for="g in categories" :key="g.schemeId" :label="g.name">
          <el-option v-for="o in g.options ?? []" :key="o.id" :value="o.id" :label="`${g.name} / ${o.name}`">
            <span :style="{ paddingLeft: `${o.level * 16}px` }">{{ o.level ? '├ ' : '' }}{{ o.name }}</span>
          </el-option>
        </el-option-group>
      </el-select>
      <p class="hint">没有单独绑定的子分类会沿用其一级分类的绑定 (显示为「继承」); 在这里单独绑定后以单独绑定为准。可同时绑定多个本站分类 (例如不同分类方案中的分类), 视频会出现在每个所选分类中; 清空选择并保存即解除绑定。
        只有绑定了本站分类的采集站分类才会采集入库; 已入库的视频在再次采集时补上新增的分类。</p>
    </AdminDialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { router, usePage } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import { Back, Plus } from '@element-plus/icons-vue'
import { http } from '../../utils/http'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import AdminDialog from '../../Components/AdminDialog.vue'
import { confirmAction } from '../../utils/confirm'
import { isCollecting, startCollect } from '../../utils/collectTasks'
import { useCan } from '../../utils/menu'

const can = useCan()

defineOptions({ layout: AdminLayout })

type BindClass = { typeId: number; typePid: number; typeName: string; categoryIds: number[] | null; categories: string[] | null; inherited?: boolean }
type CategoryGroup = { schemeId: number; name: string; options: { id: number; name: string; level: number }[] | null }

const props = defineProps<{
    source: any
    classes: BindClass[] | null
    list: any[] | null
    paging: { page: number; pageCount: number; total: number }
    query: { t: number; wd: string }
    categories: CategoryGroup[] | null
}>()
const page = usePage<{ errors?: { form?: string } }>()

const keyword = ref(props.query.wd)
watch(() => props.query.wd, (wd) => { keyword.value = wd })
const selected = ref<any[]>([])

const visit = (q: { t?: number; wd?: string; pg?: number }) => {
    router.get('/manage/collect/browse', {
        id: props.source.id,
        t: q.t ?? props.query.t,
        wd: q.wd ?? props.query.wd,
        pg: q.pg ?? props.paging.page,
    }, { preserveState: true })
}

const bindV = ref(false)
const bindForm = reactive({ typeId: 0, typeName: '', categoryIds: [] as number[] })
// 已绑定: 显示第一个本站分类名称, 多个时加上数量
const bindLabel = (c: BindClass) => {
    const names = c.categories ?? []
    if (!names.length) return '绑定'
    const first = names[0].split(' / ').pop()
    const label = names.length > 1 ? `${first} 等${names.length}个` : first
    // 没有单独绑定时沿用一级分类的绑定, 点击可单独绑定
    return c.inherited ? `继承: ${label}` : label
}
const openBind = (c: BindClass) => {
    Object.assign(bindForm, { typeId: c.typeId, typeName: c.typeName, categoryIds: [...(c.categoryIds ?? [])] })
    bindV.value = true
}
const saveBind = () => {
    router.post('/manage/collect/bind', {
        sourceId: props.source.id, typeId: bindForm.typeId, typeName: bindForm.typeName, categoryIds: bindForm.categoryIds,
    }, { preserveScroll: true, onSuccess: () => { bindV.value = false } })
}

const notify = (resp: any) => resp.data.code === 0
    ? ElMessage.success({ message: `${resp.data.msg}, 可在「采集接口」列表查看采集进度` })
    : ElMessage.error({ message: resp.data.msg })

const collectSelected = () => {
    http.post('/manage/collect/browse/collect', { id: props.source.id, ids: selected.value.map((r) => r.vod_id) }).then(notify)
}

// 采集当天 / 采全部: 弹窗显示进度 (见 Components/CollectTasks.vue)
const collectTime = async (time: number, label: string) => {
    if (time < 0 && !await confirmAction(`确认对「${props.source.name}」执行${label}? 会请求该接口全部页码, 耗时较长。`)) return
    startCollect(props.source.id, time)
}
</script>

<style scoped>
.browse-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.source-name {
  color: #606266;
}
.class-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 10px 16px;
  margin-bottom: 16px;
}
.class-item a {
  cursor: pointer;
  color: #303133;
}
.class-item a.active {
  color: #f56c6c;
}
.class-item a.bind {
  margin-left: 4px;
  color: #606266;
}
.class-item a.bind.bound {
  color: #f56c6c;
}
.class-item a.bind.inherited {
  color: #e6a23c;
}
.search-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}
.time {
  color: #f56c6c;
}
.browse-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 12px;
}
.hint {
  color: #909399;
  font-size: 12px;
  margin-top: 8px;
}
</style>
