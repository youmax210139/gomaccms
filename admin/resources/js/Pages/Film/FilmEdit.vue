<template>
  <div class="film-edit">
    <div class="head">
      <el-button :icon="Back" @click="router.visit('/manage/film/search/list')">返回列表</el-button>
      <h2>编辑视频 <span class="vod-id">ID {{ form.id }}</span></h2>
    </div>
    <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
    <el-form :model="form" label-width="90px">
      <el-card shadow="never" class="block">
        <template #header>文字</template>
        <LangTabs :languages="languages" :i18n="form.i18n" :source="form" :fields="langFields">
          <el-form-item v-for="f in textFields" :key="f.src" :label="f.label" :required="f.src === 'name'">
            <el-input v-if="f.textarea" v-model="form.content" type="textarea" :autosize="{ minRows: 4, maxRows: 12 }" />
            <el-input v-else v-model="form[f.src]" maxlength="255" />
          </el-form-item>
        </LangTabs>
      </el-card>

      <el-card shadow="never" class="block">
        <template #header>基本信息</template>
        <el-row :gutter="16">
          <el-col :md="12">
            <el-form-item label="主分类" required>
              <el-select v-model="form.cid" filterable>
                <el-option v-for="c in categories" :key="c.id" :value="c.id" :label="c.name" />
              </el-select>
              <div class="field-note">修改主分类后建议同时锁定, 否则下次采集可能把原分类加回</div>
            </el-form-item>
          </el-col>
          <el-col :md="12"><el-form-item label="首字母"><el-input v-model="form.letter" maxlength="10" /></el-form-item></el-col>
          <el-col :md="12"><el-form-item label="年份"><el-input v-model="form.year" maxlength="4" placeholder="YYYY" /></el-form-item></el-col>
          <el-col :md="12"><el-form-item label="上映日期"><el-input v-model="form.pubdate" maxlength="100" /></el-form-item></el-col>
          <el-col :md="12"><el-form-item label="视频状态"><el-input v-model="form.state" maxlength="255" placeholder="正片 / 预告片" /></el-form-item></el-col>
          <el-col :md="12"><el-form-item label="人气"><el-input-number v-model="form.hits" :min="0" :step="1" step-strictly /></el-form-item></el-col>
          <el-col :md="12"><el-form-item label="评分"><el-input-number v-model="form.score" :min="0" :max="10" :step="0.1" :precision="1" /></el-form-item></el-col>
          <el-col :md="12"><el-form-item label="推荐"><el-input-number v-model="form.level" :min="0" :max="9" :step="1" step-strictly /></el-form-item></el-col>
          <el-col :md="12">
            <el-form-item label="审核"><el-switch v-model="form.status" :active-value="1" :inactive-value="0" inline-prompt active-text="已审核" inactive-text="未审核" /></el-form-item>
          </el-col>
          <el-col :md="12">
            <el-form-item label="锁定">
              <el-switch v-model="form.lock" :active-value="1" :inactive-value="0" inline-prompt active-text="锁定" inactive-text="未锁" />
              <span class="field-tip">锁定后采集不再更新此视频 (备注、播放地址等)</span>
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="封面">
              <div class="pic-row">
                <el-image v-if="form.pic" :src="form.pic" fit="cover" class="pic" :preview-src-list="[form.pic]" preview-teleported />
                <el-input v-model="form.pic" maxlength="1024" placeholder="图片 URL" />
                <el-upload :show-file-list="false" action="#" :http-request="upload"><el-button>上传</el-button></el-upload>
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <el-card shadow="never" class="block">
        <template #header>
          <div class="card-head">
            <span>播放组</span>
            <el-button size="small" :icon="Plus" @click="form.playGroups.push({ from: '', text: '' })">添加播放组</el-button>
          </div>
        </template>
        <p class="tip">每行一集: 集名$地址 (也可以只填地址)。没有地址的播放组不会保存。</p>
        <div v-for="(g, i) in form.playGroups" :key="i" class="group">
          <div class="group-head">
            <el-select v-model="g.from" filterable allow-create placeholder="播放器" class="group-from">
              <el-option v-for="p in players" :key="p.code" :value="p.code" :label="`${p.name} (${p.code})`" />
            </el-select>
            <span class="field-tip">{{ episodes(g.text) }} 集</span>
            <el-button size="small" :icon="Delete" @click="form.playGroups.splice(i, 1)">删除</el-button>
          </div>
          <el-input v-model="g.text" type="textarea" :autosize="{ minRows: 3, maxRows: 10 }" />
        </div>
      </el-card>

      <div class="actions">
        <el-button type="primary" :loading="form.processing" @click="save">保存</el-button>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { Back, Delete, Plus } from '@element-plus/icons-vue'
import { computed, watch } from 'vue'
import { router, useForm, usePage } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import LangTabs, { type LangField } from '../../Components/LangTabs.vue'
import { i18nOf } from '../../utils/i18n'
import { http } from '../../utils/http'

type VodText = { name: string; sub: string; content: string; actor: string; director: string; writer: string; remarks: string; area: string; langText: string; class: string }
type Vod = {
  id: number; cid: number; name: string; sub: string; letter: string; class: string; pic: string; actor: string; director: string
  writer: string; remarks: string; pubdate: string; area: string; lang: string; year: string; state: string; content: string
  status: number; level: number; lock: number; hits: number; score: number
  playGroups: { from: string; text: string }[] | null; i18n: Record<string, VodText> | null; base: string
}

const props = defineProps<{
  vod: Vod
  categories: { id: number; name: string }[] | null
  players: { code: string; name: string }[] | null
  languages: { code: string; name: string }[] | null
}>()
const page = usePage<{ errors?: { form?: string } }>()
defineOptions({ layout: AdminLayout })

const categories = computed(() => props.categories ?? [])
const players = computed(() => props.players ?? [])
const languages = computed(() => props.languages ?? [])

// 可翻译的栏位: key 为译文栏位, src 为原文栏位
const textFields = [
  { key: 'name', src: 'name', label: '片名' },
  { key: 'sub', src: 'sub', label: '别名' },
  { key: 'class', src: 'class', label: '剧情标签' },
  { key: 'actor', src: 'actor', label: '主演' },
  { key: 'director', src: 'director', label: '导演' },
  { key: 'writer', src: 'writer', label: '编剧' },
  { key: 'area', src: 'area', label: '地区' },
  { key: 'langText', src: 'lang', label: '语言' },
  { key: 'remarks', src: 'remarks', label: '备注' },
  { key: 'content', src: 'content', label: '简介', textarea: true },
] as const
// 译文页签: 简介 4 行起, 其余单行 255 字
const langFields: LangField[] = textFields.map((f) => ({ ...f, ...('textarea' in f ? { rows: 4 } : { maxlength: 255 }) }))

const emptyText = (): VodText => ({ name: '', sub: '', content: '', actor: '', director: '', writer: '', remarks: '', area: '', langText: '', class: '' })
const formData = (v: Vod) => ({ ...v, playGroups: v.playGroups ?? [], i18n: i18nOf(languages.value, v.i18n, emptyText) })
const form = useForm(formData(props.vod))
// 保存后页面不会重新挂载 (preserveState): 用服务器整理后的数据 (含新的 base 指纹) 重设表单
watch(() => props.vod, (v) => Object.assign(form, formData(v)))

const episodes = (text: string) => text.split('\n').filter((l) => l.trim()).length

const upload = async (options: any) => {
  const data = new FormData()
  data.append('file', options.file)
  const resp = await http.post('/manage/file/upload', data)
  if (resp.data.code === 0) form.pic = resp.data.data
  else ElMessage.error({ message: resp.data.msg })
}

const save = () => {
  // 成功 / 失败提示由 app.ts 的全局处理显示
  // 带上 id: 保存失败重新渲染时地址栏仍是这部视频的编辑页, 重新载入不会回到列表
  form.post(`/manage/film/edit?id=${form.id}`, { preserveScroll: true })
}
</script>

<style scoped>
.head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.head h2 {
  margin: 0;
  font-size: 18px;
}
.vod-id {
  color: #909399;
  font-size: 13px;
  font-weight: normal;
}
.block {
  margin-bottom: 16px;
}
.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.tip {
  margin: 0 0 10px;
  color: #909399;
  font-size: 12px;
}
.field-note {
  width: 100%;
  margin-top: 4px;
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
}
.field-tip {
  margin-left: 8px;
  color: #909399;
  font-size: 12px;
}
.pic-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.pic {
  width: 48px;
  height: 64px;
  flex: none;
  border-radius: 4px;
}
.group {
  margin-bottom: 14px;
}
.group-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.group-from {
  width: 240px;
}
.actions {
  display: flex;
  justify-content: center;
  margin: 8px 0 24px;
}
</style>
