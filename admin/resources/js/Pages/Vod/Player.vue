<template>
  <div>
    <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
    <p class="tip">每个播放组代码 (采集站返回的 vod_play_from, 如 bfzym3u8) 对应一个播放器, 决定前台播放线路的名称与顺序; 停用后前台不显示该线路。
      采集到新的播放组代码时会自动新增, 名称默认为采集接口名称。</p>
    <TableToolbar>
      <el-button type="primary" :icon="Plus" @click="openAdd">添加</el-button>
    </TableToolbar>
    <el-table stripe :data="players" size="default" table-layout="auto" empty-text="暂无播放器, 采集后会自动新增">
      <el-table-column prop="code" label="代码" min-width="140"/>
      <el-table-column prop="name" label="名称 (前台线路名称)" min-width="160"/>
      <el-table-column label="视频数" align="center" width="100">
        <template #default="{ row }">{{ counts[row.code] ?? 0 }}</template>
      </el-table-column>
      <el-table-column label="排序" align="center" width="90">
        <template #default="{ row }">{{ row.sort }}</template>
      </el-table-column>
      <el-table-column label="状态" align="center" width="100">
        <template #default="{ row }">
          <el-switch :model-value="row.status" inline-prompt active-text="启用" inactive-text="停用"
                     @change="(v: string | number | boolean) => changeStatus(row, Boolean(v))"/>
        </template>
      </el-table-column>
      <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip/>
      <el-table-column label="操作" align="center" width="160">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" @click="delPlayer(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog v-model="dialog" width="520px" :title="form.isNew ? '添加播放器' : '编辑播放器'" :loading="form.processing" @confirm="save">
      <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
      <el-form :model="form" label-width="90px">
        <el-form-item label="代码" required>
          <el-input v-model="form.code" :disabled="!form.isNew" maxlength="60" placeholder="与采集站的播放组代码一致, 如 bfzym3u8"/>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.name" maxlength="60" placeholder="前台显示的线路名称, 如 高清线路"/>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" :step="1" step-strictly/> <span class="field-tip">越小越靠前</span>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" inline-prompt active-text="启用" inactive-text="停用"/>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" maxlength="255" show-word-limit/>
        </el-form-item>
      </el-form>
    </AdminDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { router, useForm, usePage } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import AdminDialog from '../../Components/AdminDialog.vue'
import TableToolbar from '../../Components/TableToolbar.vue'
import { confirmAction } from '../../utils/confirm'

type Player = { code: string; name: string; status: boolean; sort: number; remark: string }

const props = defineProps<{ players: Player[] | null; counts: Record<string, number> | null }>()
const page = usePage<{ errors?: { form?: string } }>()

defineOptions({ layout: AdminLayout })

const players = computed(() => props.players ?? [])
const counts = computed(() => props.counts ?? {})

const dialog = ref(false)
const form = useForm({ code: '', name: '', status: true, sort: 0, remark: '', isNew: true })
const openAdd = () => {
    Object.assign(form, { code: '', name: '', status: true, sort: 0, remark: '', isNew: true })
    dialog.value = true
}
const openEdit = (p: Player) => {
    Object.assign(form, { ...p, isNew: false })
    dialog.value = true
}
const save = () => {
    if (!form.code.trim() || !form.name.trim()) {
        ElMessage.warning('代码与名称不能为空')
        return
    }
    form.post('/manage/film/player/save', {
        preserveScroll: true,
        onSuccess: () => { if (!page.props.errors?.form) dialog.value = false },
    })
}
const changeStatus = (p: Player, status: boolean) => {
    router.post('/manage/film/player/state', { code: p.code, status }, { preserveScroll: true })
}
const delPlayer = async (p: Player) => {
    if (!await confirmAction(`确认删除播放器「${p.name}」(${p.code})? 视频的播放地址不会删除, 前台线路名称将改为显示代码。`)) return
    router.get('/manage/film/player/del', { code: p.code }, { preserveScroll: true })
}
</script>

<style scoped>
.tip {
  color: #909399;
  font-size: 13px;
  margin: 0 0 12px;
  line-height: 1.6;
}
.field-tip {
  margin-left: 8px;
  color: #909399;
  font-size: 12px;
}
</style>
