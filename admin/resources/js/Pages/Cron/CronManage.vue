<template>
  <div>
    <p v-if="page.props.errors?.form" style="color: #f56c6c;">{{ page.props.errors.form }}</p>
    <TableToolbar>
      <el-button type="primary" :icon="Clock" @click="openAddDialog">创建定时任务</el-button>
    </TableToolbar>
    <el-table stripe :data="list" style="width: 100%" size="default" table-layout="auto">
      <el-table-column prop="id" label="任务ID">
        <template #default="scope"><el-tag disable-transitions>{{ scope.row.id }}</el-tag></template>
      </el-table-column>
      <el-table-column prop="remark" label="任务描述" />
      <el-table-column prop="model" align="center" label="任务类型">
        <template #default="scope">
          <el-tag disable-transitions>{{ scope.row.model == 0 ? '自动更新' : scope.row.model == 1 ? '自定义任务' : '已废弃' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="state" align="center" label="是否启用">
        <template #default="scope">
          <el-switch v-model="scope.row.state" @change="changeTaskState(scope.row.id, scope.row.state)" inline-prompt active-text="启用" inactive-text="禁用"/>
        </template>
      </el-table-column>
      <el-table-column prop="preV" align="center" label="上次执行时间">
        <template #default="scope"><el-tag type="success" disable-transitions>{{ scope.row.preV }}</el-tag></template>
      </el-table-column>
      <el-table-column prop="next" align="center" label="下次执行时间">
        <template #default="scope"><el-tag type="warning" disable-transitions>{{ scope.row.next }}</el-tag></template>
      </el-table-column>
      <el-table-column label="操作" align="center" min-width="140">
        <template #default="scope">
          <el-button size="small" @click="openEditDialog(scope.row.id)">编辑</el-button>
          <el-button size="small" @click="delTask(scope.row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog v-model="dialogV.addV" title="创建定时任务" confirm-text="创建" :loading="addForm.processing" @confirm="submitAdd">
      <el-form :model="addForm">
        <el-form-item label="任务周期">
          <el-input v-model="addForm.spec" placeholder="定时任务Cron表达式 (例: [0 */20 * * * ?] 每20分钟执行一次)"/>
        </el-form-item>
        <el-form-item label="任务描述">
          <el-input v-model="addForm.remark" placeholder="定时任务描述信息"/>
        </el-form-item>
        <el-form-item label="任务类型">
          <el-radio-group v-model="addForm.model">
            <el-tooltip effect="dark" content="执行所有已启用站点的采集任务" placement="top">
              <el-radio :label="0">自动更新</el-radio>
            </el-tooltip>
            <el-tooltip effect="dark" content="只执行指定站点的采集任务" placement="top">
              <el-radio :label="1">自定义更新</el-radio>
            </el-tooltip>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="addForm.model == 1" label="资源绑定">
          <el-select v-model="addForm.ids" multiple collapse-tags collapse-tags-tooltip placeholder="Select" style="width: 240px">
            <el-option v-for="item in options" :key="item.id" :label="item.name" :value="item.id"/>
          </el-select>
        </el-form-item>
        <el-form-item label="采集时长">
          <el-tooltip effect="dark" content="采集最近x小时更新的视频,负数则默认采集所有资源" placement="top">
            <el-input-number v-model="addForm.time" :step="1" step-strictly />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="任务状态">
          <el-switch v-model="addForm.state" inline-prompt active-text="开启" inactive-text="禁用"/>
        </el-form-item>
      </el-form>
    </AdminDialog>

    <AdminDialog v-model="dialogV.editV" title="编辑定时任务" :loading="editForm.processing" @confirm="submitEdit">
      <el-form :model="editForm">
        <el-form-item label="任务标识"><el-tag type="success" disable-transitions>{{ editForm.id }}</el-tag></el-form-item>
        <el-form-item label="任务描述"><el-input v-model="editForm.remark" placeholder="定时任务描述信息"/></el-form-item>
        <el-form-item label="任务周期"><el-tag disable-transitions>{{ editForm.spec }}</el-tag></el-form-item>
        <el-form-item label="任务类型">
          <el-tag disable-transitions>{{ editForm.model == 0 ? '自动更新' : editForm.model == 1 ? '自定义更新' : '已废弃' }}</el-tag>
        </el-form-item>
        <el-form-item v-if="editForm.model == 1" label="资源绑定">
          <el-select v-model="editForm.ids" multiple collapse-tags collapse-tags-tooltip placeholder="Select" style="width: 240px">
            <el-option v-for="item in options" :key="item.id" :label="item.name" :value="item.id"/>
          </el-select>
        </el-form-item>
        <el-form-item label="采集时长">
          <el-tooltip effect="dark" content="采集最近x小时更新的视频,负数则默认采集所有资源" placement="top">
            <el-input-number v-model="editForm.time" :step="1" step-strictly />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="任务状态"><el-switch v-model="editForm.state" inline-prompt active-text="开启" inactive-text="禁用"/></el-form-item>
      </el-form>
    </AdminDialog>
  </div>
</template>

<script setup lang="ts">
import { Clock } from '@element-plus/icons-vue'
import { computed, reactive } from 'vue'
import { router, useForm, usePage } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import { http } from '../../utils/http'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import AdminDialog from '../../Components/AdminDialog.vue'
import { confirmAction } from '../../utils/confirm'
import TableToolbar from '../../Components/TableToolbar.vue'

const props = defineProps<{ list: any[] }>()
const page = usePage<{ errors?: { form?: string } }>()

defineOptions({ layout: AdminLayout })

const list = computed(() => props.list)
const options = reactive<{ id: string; name: string }[]>([])
const dialogV = reactive({ addV: false, editV: false })

const emptyAdd = () => ({ spec: '', remark: '', model: 1, ids: [] as string[], time: 0, state: false })
const emptyEdit = () => ({ id: '', spec: '', remark: '', model: 1, ids: [] as string[], time: 0, state: false })

const addForm = useForm(emptyAdd())
const editForm = useForm(emptyEdit())

const loadOptions = () => {
    http.get('/manage/collect/options').then((resp: any) => {
        if (resp.data.code === 0) {
            options.splice(0, options.length, ...(resp.data.data ?? []))
        } else {
            ElMessage.error({ message: resp.data.msg })
        }
    })
}

const openAddDialog = () => {
    Object.assign(addForm, emptyAdd())
    loadOptions()
    dialogV.addV = true
}
const submitAdd = () => {
    addForm.post('/manage/cron/add', { onSuccess: () => { dialogV.addV = false } })
}

const openEditDialog = (id: string) => {
    loadOptions()
    http.get('/manage/cron/find', { params: { id } }).then((resp: any) => {
        if (resp.data.code === 0) {
            Object.assign(editForm, resp.data.data)
            dialogV.editV = true
        } else {
            ElMessage.error({ message: resp.data.msg })
        }
    })
}
const submitEdit = () => {
    editForm.post('/manage/cron/update', { onSuccess: () => { dialogV.editV = false } })
}

const delTask = async (row: any) => {
    if (!await confirmAction(`确认删除定时任务「${row.remark || row.id}」?`)) return
    router.get('/manage/cron/del', { id: row.id })
}

const changeTaskState = (id: string, state: boolean) => {
    router.post('/manage/cron/change', { id, state })
}
</script>
