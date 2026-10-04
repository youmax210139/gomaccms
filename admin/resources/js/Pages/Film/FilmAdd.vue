<template>
  <div>
    <h2 style="text-align: start">添加视频</h2>
    <p v-if="page.props.errors?.form" style="color: #f56c6c;">{{ page.props.errors.form }}</p>
    <el-form :model="form" class="film_add_form">
      <el-form-item>
        <div class="el-input-group__prepend" style="border: 1px solid #dcdfe6; border-right: none; border-radius: 3px; height: 32px">视频分类: </div>
        <el-select v-model="currentClass" style="width: calc(100% - 103px)" @change="changeClass" placeholder="视频分类选择">
          <el-option v-for="item in category" :key="item.id" :label="item.name" :value="item.id"/>
        </el-select>
      </el-form-item>
      <el-form-item><el-input v-model="form.name" placeholder="请输入视频名称" clearable><template #prepend>视频名称: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.subTitle" placeholder="视频别名, 可留空" clearable><template #prepend>视频别名: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.initial" placeholder="视频检索首字母, 大写" clearable><template #prepend>首字母: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.classTag" placeholder="视频剧情标签(多标签以逗号分隔): 奇幻,校园,爱情" clearable><template #prepend>剧情Tag: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.director" placeholder="导演名, 多个名称以逗号进行分隔" clearable><template #prepend>导演: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.actor" placeholder="主演名, 多个名称以逗号进行分隔" clearable><template #prepend>主演: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.writer" placeholder="作者名, 多个名称以逗号进行分隔" clearable><template #prepend>作者: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.remarks" placeholder="视频更新进度信息, 完结, HD, 更新至xx集" clearable><template #prepend>更新状态: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.releaseDate" placeholder="视频上映时间: YYYY-MM-DD" clearable><template #prepend>上映时间: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.area" placeholder="视频来源地区信息" clearable><template #prepend>地区: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.lang" placeholder="视频语言信息" clearable><template #prepend>语言: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.year" placeholder="视频上映年份信息: YYYY" clearable><template #prepend>年份: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.state" placeholder=" 视频状态: 正片 | 预告片" clearable><template #prepend>视频状态: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.dbId" placeholder="豆瓣ID" clearable><template #prepend>豆瓣Id: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.dbScore" placeholder="豆瓣评分" clearable><template #prepend>豆瓣评分: </template></el-input></el-form-item>
      <el-form-item><el-input v-model="form.hits" placeholder="视频热度(播放数)" clearable><template #prepend>视频热度: </template></el-input></el-form-item>
      <el-form-item>
        <el-input v-model="form.picture" placeholder="输入图片URL链接或点击上传到服务器并自动生成URL连接信息)" clearable>
          <template #prepend>视频海报: </template>
          <template #append>
            <el-upload class="upload-demo" :show-file-list="false" action="#" :http-request="customUpload">
              <el-button type="primary">上传图片</el-button>
            </el-upload>
          </template>
        </el-input>
      </el-form-item>
      <el-form-item><el-input v-model="form.playForm" placeholder="视频播放资源来源: xxXm3u8" clearable><template #prepend>播放来源: </template></el-input></el-form-item>
      <el-form-item>
        <template #label><span class="el-input-group__prepend cus_label">剧情简介: </span></template>
        <el-input v-model="form.content" :autosize="{ minRows: 2, maxRows: 5 }" type="textarea" placeholder="视频剧情描述信息" />
      </el-form-item>
      <el-form-item>
        <template #label><span class="el-input-group__prepend cus_label">播放地址: </span></template>
        <el-input v-model="form.playLink" :autosize="{ minRows: 2, maxRows: 5 }" type="textarea"
                  placeholder="视频播放地址信息: &#10;格式: 第01集$https://xxx/xxx/index.m3u8#第02集$https://xxx/xxx/index.m3u8" />
      </el-form-item>
      <el-form-item class="form_btn">
        <el-button type="primary" :loading="form.processing" @click="addFilm">添加视频</el-button>
        <el-button @click="resetForm">清空信息</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useForm, usePage } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import { http } from '../../utils/http'
import AdminLayout from '../../Layouts/AdminLayout.vue'

const props = defineProps<{ category: { id: number; pid: number; name: string }[] }>()
const page = usePage<{ errors?: { form?: string } }>()

defineOptions({ layout: AdminLayout })

const category = props.category

const formInit = () => ({
    id: 0, cid: 0, pid: 0, name: '', picture: '', subTitle: '', cName: '', enName: '', initial: '',
    classTag: '', actor: '', director: '', writer: '', content: '', remarks: '', releaseDate: '',
    area: '', lang: '', year: '', state: '', updateTime: '', addTime: '', dbId: 0, dbScore: '',
    hits: 0, playForm: '', playLink: '',
})

const form = useForm(formInit())
const currentClass = ref(0)

const changeClass = (value: number) => {
    const item = category.find((c) => c.id === value)
    if (item) {
        form.cid = item.id as any
        form.pid = item.pid as any
        form.cName = item.name
    }
}

const customUpload = (options: any) => {
    const formData = new FormData()
    formData.append('file', options.file)
    http.post('/manage/file/upload', formData).then((resp: any) => {
        if (resp.data.code === 0) {
            ElMessage.success({ message: resp.data.msg })
            form.picture = resp.data.data
        } else {
            ElMessage.error({ message: resp.data.msg })
        }
    })
}

const addFilm = () => {
    form.transform((data) => ({
        ...data,
        dbId: Number(data.dbId) || 0,
        hits: Number(data.hits) || 0,
    })).post('/manage/film/add')
}

const resetForm = () => {
    Object.assign(form, formInit())
}
</script>

<style scoped>
.film_add_form {
  width: 100%;
  flex-flow: wrap;
  display: flex;
  justify-content: start;
}
:deep(.el-form-item) {
  width: calc(50% - 120px);
  margin: 15px 60px;
}
.form_btn {
  width: 100% !important;
  margin: 40px auto;
}
:deep(.form_btn .el-form-item__content) {
  justify-content: center;
}
:deep(.el-form-item__label) {
  padding-right: 0 !important;
}
.cus_label {
  border: 1px solid #dcdfe6;
  border-right: none;
  border-radius: 3px;
}
</style>
