<template>
  <el-divider content-position="left">站点设定 (不分语言)</el-divider>
  <el-form-item label="网站 Logo">
    <div class="logo-field">
      <el-tooltip content="点击从图库选择" placement="top">
        <el-image v-if="info.logo" class="logo-preview logo-pick" :src="info.logo" fit="contain" @click="pickerV = true"/>
        <div v-else class="logo-empty logo-pick" @click="pickerV = true">从图库选择</div>
      </el-tooltip>
      <div class="logo-actions">
        <el-upload :show-file-list="false" accept="image/*" :http-request="uploadLogo">
          <el-button :loading="logoUploading">{{ info.logo ? '更换 Logo' : '上传 Logo' }}</el-button>
        </el-upload>
        <el-button v-if="info.logo" type="danger" plain @click="info.logo = ''">清除</el-button>
      </div>
    </div>
    <GalleryPicker v-model:visible="pickerV" :model-value="info.logo" @select="(link) => (info.logo = link)"/>
  </el-form-item>
  <el-form-item label="客服 Email">
    <el-input v-model="info.serviceEmail" maxlength="100" placeholder="service@example.com"/>
  </el-form-item>
  <el-form-item label="统计代码">
    <el-input v-model="info.analyticsCode" type="textarea" :autosize="{ minRows: 3, maxRows: 10 }" placeholder="原样输出到前台页面底部, 如 <script>...</script>"/>
  </el-form-item>
  <slot name="info"/>
  <slot name="status"/>
</template>

<script lang="ts">
// 站点信息在某个语言的文字 (Logo、Email、统计代码不分语言)
export type SiteText = { siteName: string; seoTitle: string; keyword: string; describe: string; legalInfo: string }
export const emptySiteText = (): SiteText => ({ siteName: '', seoTitle: '', keyword: '', describe: '', legalInfo: '' })
</script>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, type UploadRequestOptions } from 'element-plus'
import { http } from '../utils/http'
import GalleryPicker from './GalleryPicker.vue'


export type SiteInfo = {
  siteName: string
  logo: string
  seoTitle: string
  keyword: string
  describe: string
  serviceEmail: string
  analyticsCode: string
  legalInfo: string
  i18n: Record<string, SiteText>
}

// 不分语言的站点设定 (Logo / 客服 Email / 统计代码); 网站名称、SEO、法律信息在 SiteTextFields (SEO 弹窗) 中编辑.
// 直接修改传入的表单对象 (与 BannerForm 相同的用法)
const props = defineProps<{ info: SiteInfo }>()

const pickerV = ref(false)
const logoUploading = ref(false)
const uploadLogo = async (options: UploadRequestOptions) => {
    const formData = new FormData()
    formData.append('file', options.file)
    logoUploading.value = true
    try {
        const resp = await http.post('/manage/file/upload', formData)
        if (resp.data.code === 0) {
            props.info.logo = resp.data.data
        } else {
            ElMessage.error(resp.data.msg || 'Logo 上传失败')
        }
    } catch {
        ElMessage.error('Logo 上传失败')
    } finally {
        logoUploading.value = false
    }
}
</script>

<style scoped>
.logo-field {
  display: flex;
  align-items: center;
  gap: 16px;
}
.logo-preview,
.logo-empty {
  width: 120px;
  height: 60px;
  border: 1px dashed #dcdfe6;
  border-radius: 4px;
}
.logo-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #909399;
  font-size: 13px;
}
.logo-pick {
  cursor: pointer;
}
.logo-pick:hover {
  border-color: var(--el-color-primary);
}
.logo-actions {
  display: flex;
  gap: 8px;
}
</style>
