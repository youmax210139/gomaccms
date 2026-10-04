<template>
  <div>
    <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
    <p class="tip">前台可用的语言。每个分类方案在「分类管理 → 修改方案」中选择默认语言与启用的语言, 访客可在前台切换该方案启用的语言。
      中文 ({{ sourceLang }}) 是采集内容的原文语言, 不能停用; 翻译服务代码供影片机器翻译使用 (LibreTranslate 的语言代码, 如 zh、vi)。</p>
    <TableToolbar>
      <el-button type="primary" :icon="Plus" @click="openAdd">添加</el-button>
    </TableToolbar>
    <el-table stripe :data="languages" size="default" table-layout="auto" empty-text="暂无语言">
      <el-table-column prop="code" label="代码" min-width="120"/>
      <el-table-column prop="name" label="名称 (前台显示)" min-width="160"/>
      <el-table-column prop="libreCode" label="翻译服务代码" min-width="120"/>
      <el-table-column label="排序" align="center" width="90">
        <template #default="{ row }">{{ row.sort }}</template>
      </el-table-column>
      <el-table-column label="状态" align="center" width="100">
        <template #default="{ row }">
          <el-switch :model-value="row.enabled" :disabled="row.code === sourceLang" inline-prompt active-text="启用" inactive-text="停用"
                     @change="(v: string | number | boolean) => changeEnabled(row, Boolean(v))"/>
        </template>
      </el-table-column>
      <el-table-column label="操作" align="center" width="100">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog v-model="dialog" width="520px" :title="form.isNew ? '添加语言' : '编辑语言'" :loading="form.processing" @confirm="save">
      <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
      <el-form :model="form" label-width="110px">
        <el-form-item label="代码" required>
          <el-input v-model="form.code" :disabled="!form.isNew" maxlength="16" placeholder="如 vi、en、zh-TW (即 html lang)"/>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.name" maxlength="40" placeholder="前台切换显示的名称, 如 Tiếng Việt"/>
        </el-form-item>
        <el-form-item label="翻译服务代码">
          <el-input v-model="form.libreCode" maxlength="16" placeholder="LibreTranslate 的语言代码, 如 vi"/>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" :step="1" step-strictly/> <span class="field-tip">越小越靠前</span>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.enabled" :disabled="form.code === sourceLang" inline-prompt active-text="启用" inactive-text="停用"/>
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

type Language = { code: string; name: string; libreCode: string; enabled: boolean; sort: number }

const props = defineProps<{ languages: Language[] | null; sourceLang: string }>()
const page = usePage<{ errors?: { form?: string } }>()

defineOptions({ layout: AdminLayout })

const languages = computed(() => props.languages ?? [])
const sourceLang = computed(() => props.sourceLang)

const dialog = ref(false)
const form = useForm({ code: '', name: '', libreCode: '', enabled: true, sort: 0, isNew: true })
const openAdd = () => {
    Object.assign(form, { code: '', name: '', libreCode: '', enabled: true, sort: 0, isNew: true })
    dialog.value = true
}
const openEdit = (l: Language) => {
    Object.assign(form, { ...l, isNew: false })
    dialog.value = true
}
const save = () => {
    if (!form.code.trim() || !form.name.trim()) {
        ElMessage.warning('代码与名称不能为空')
        return
    }
    form.post('/manage/lang/save', {
        preserveScroll: true,
        onSuccess: () => { if (!page.props.errors?.form) dialog.value = false },
    })
}
const changeEnabled = (l: Language, enabled: boolean) => {
    router.post('/manage/lang/state', { code: l.code, enabled }, { preserveScroll: true })
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
