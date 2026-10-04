<template>
  <el-form :model="banner" label-width="90px">
    <el-form-item label="绑定视频">
      <div class="vod-field">
        <template v-if="banner.mid">
          <el-tag disable-transitions>ID {{ banner.mid }}</el-tag>
          <span class="vod-name">{{ vodName || '' }}</span>
        </template>
        <span v-else class="field-tip no-tip-margin">未绑定 (海报不链接)</span>
        <el-button size="small" @click="pickerVisible = true">{{ banner.mid ? '更换' : '选择视频' }}</el-button>
        <el-button v-if="banner.mid" size="small" @click="unbind">解除</el-button>
      </div>
    </el-form-item>
    <el-form-item label="视频海报">
      <el-input v-model="banner.poster" placeholder="视频海报访问URL"/>
      <input type="file" @change="onFile($event, 'poster')" />
    </el-form-item>
    <el-form-item label="视频封面">
      <el-input v-model="banner.picture" placeholder="视频封面访问URL"/>
      <input type="file" @change="onFile($event, 'picture')" />
    </el-form-item>
    <el-form-item label="排序分值">
      <el-input-number v-model="banner.sort" :min="0" :step="1" step-strictly/> <span class="field-tip">越小越靠前</span>
    </el-form-item>
    <el-form-item label="状态">
      <el-switch v-model="banner.status" inline-prompt active-text="启用" inactive-text="禁用"/>
    </el-form-item>
    <LangTabs :languages="languages" :i18n="banner.i18n" :source="banner" :fields="bannerFields">
      <el-form-item label="海报名称" required>
        <el-input v-model="banner.name" maxlength="255" placeholder="海报名称"/>
      </el-form-item>
    </LangTabs>
    <VodPicker v-model:visible="pickerVisible" :scheme="scheme" :selected="banner.mid" @select="onPick"/>
  </el-form>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import LangTabs, { type LangField } from './LangTabs.vue'
import VodPicker, { type PickedVod } from './VodPicker.vue'

export type BannerText = { name: string }
export type BannerData = {
  mid: number; name: string; poster: string; picture: string; sort: number
  status: boolean; i18n: Record<string, BannerText>
}

// scheme: 海报所属的分类方案 (只能绑定该方案中的视频); vodName: 已绑定视频的名称
const props = defineProps<{ banner: BannerData; languages: { code: string; name: string }[]; scheme: number; vodName?: string }>()

const emit = defineEmits<{ upload: [file: File, field: 'poster' | 'picture']; bind: [name: string] }>()

const bannerFields: LangField[] = [{ key: 'name', label: '海报名称', maxlength: 255 }]

const pickerVisible = ref(false)
// 选中视频: 绑定其 ID; 海报名称与封面为空时用视频的名称与封面
const onPick = (v: PickedVod) => {
  props.banner.mid = v.id
  if (!props.banner.name.trim()) props.banner.name = v.name
  if (!props.banner.picture.trim()) props.banner.picture = v.picture
  emit('bind', v.name)
}
const unbind = () => {
  props.banner.mid = 0
  emit('bind', '')
}

const onFile = (e: Event, field: 'poster' | 'picture') => {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (file) emit('upload', file, field)
}
</script>

<style scoped>
.field-tip {
  margin-left: 8px;
  color: #909399;
  font-size: 12px;
}
.no-tip-margin {
  margin-left: 0;
}
.vod-field {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.vod-name {
  color: var(--el-text-color-regular);
}
</style>
