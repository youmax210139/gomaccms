<template>
  <div class="collect-form">
    <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
    <el-form :model="form" label-width="110px">
      <el-divider content-position="left">基本信息</el-divider>
      <el-form-item label="资源名称" required>
        <el-input v-model="form.name" maxlength="20" placeholder="自定义资源名称(禁用汉字)"/>
      </el-form-item>
      <el-form-item label="接口地址" required>
        <el-input v-model="form.uri" placeholder="如 https://bfzyapi.com/api.php/provide/vod/"/>
      </el-form-item>
      <el-form-item label="附加参数">
        <el-input v-model="form.params" placeholder="&ct=1"/>
        <div class="hint">提示信息: 一般&开头, 例如老版xml格式采集下载地址需加入&ct=1</div>
      </el-form-item>

      <el-divider content-position="left">接口设置</el-divider>
      <el-form-item label="接口类型">
        <el-radio-group v-model="form.resultModel">
          <el-radio :value="1">xml</el-radio>
          <el-radio :value="0">json</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="资源类型">
        <el-radio-group v-model="form.collectType">
          <el-radio v-for="(t, i) in collectTypes" :key="i" :value="i">{{ t }}</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="采集间隔">
        <el-input-number v-model="form.interval" :min="0" :step="100" step-strictly/>
        <span class="hint inline">单次采集请求的时间间隔, 单位 ms, 0 为不限制</span>
      </el-form-item>
      <el-form-item label="是否启用">
        <el-switch v-model="form.state" inline-prompt active-text="启用" inactive-text="禁用"/>
      </el-form-item>

      <el-divider content-position="left">数据操作</el-divider>
      <el-form-item label="数据操作">
        <el-radio-group v-model="form.operation">
          <el-radio :value="0">新增+更新</el-radio>
          <el-radio :value="1">新增</el-radio>
          <el-radio :value="2">更新</el-radio>
        </el-radio-group>
        <div class="hint">提示信息: 如果某个资源作为副资源不想新增数据, 可以只勾选更新。</div>
      </el-form-item>
      <el-form-item label="地址过滤">
        <el-radio-group v-model="form.filterMode">
          <el-radio :value="0">不过滤</el-radio>
          <el-radio :value="1">新增+更新</el-radio>
          <el-radio :value="2">新增</el-radio>
          <el-radio :value="3">更新</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="过滤代码">
        <el-input v-model="form.filterCode" placeholder="多组地址的资源开启白名单后只会入库指定代码的地址。比如 youku,iqiyi"/>
      </el-form-item>
      <el-form-item label="过滤年份">
        <el-input v-model="form.filterYear" placeholder="填写后仅入库指定年份的视频。多个年份用英文半角逗号分隔, 如 2022,2023"/>
      </el-form-item>

      <el-divider content-position="left">图片</el-divider>
      <el-form-item label="同步图片">
        <el-radio-group v-model="form.syncImage">
          <el-radio :value="0">跟随全局</el-radio>
          <el-radio :value="1">开启</el-radio>
          <el-radio :value="2">关闭</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item>
        <el-button type="success" :loading="testing" @click="apiTest">测试</el-button>
        <el-button type="primary" :loading="form.processing" @click="save">保存</el-button>
        <el-button @click="form.reset()">还原</el-button>
        <el-button link @click="router.get('/manage/collect/list')">返回列表</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { router, useForm, usePage } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import { http } from '../../utils/http'
import AdminLayout from '../../Layouts/AdminLayout.vue'

defineOptions({ layout: AdminLayout })

const props = defineProps<{ source: any }>()
const page = usePage<{ errors?: { form?: string } }>()

const collectTypes = ['视频', '文章', '演员', '角色', '网站']

// 表单初始值即页面载入时的数据, 「还原」回到这里
const form = useForm({
    id: props.source.id ?? '',
    name: props.source.name ?? '',
    uri: props.source.uri ?? '',
    params: props.source.params ?? '',
    resultModel: props.source.resultModel ?? 0,
    collectType: props.source.collectType ?? 0,
    interval: props.source.interval ?? 0,
    state: props.source.state ?? true,
    operation: props.source.operation ?? 0,
    filterMode: props.source.filterMode ?? 0,
    filterCode: props.source.filterCode ?? '',
    filterYear: props.source.filterYear ?? '',
    syncImage: props.source.syncImage ?? 0,
})


// 测试: 用当前表单的接口地址/附加参数/接口类型请求一页数据, 不写入数据
const testing = ref(false)
const apiTest = () => {
    testing.value = true
    http.post('/manage/collect/test', form.data()).then((resp: any) => {
        ElMessage[resp.data.code === 0 ? 'success' : 'error']({ message: resp.data.msg })
    }).finally(() => { testing.value = false })
}

const save = () => {
    form.post(form.id ? '/manage/collect/update' : '/manage/collect/add')
}
</script>

<style scoped>
.collect-form {
  max-width: 900px;
}
.hint {
  width: 100%;
  color: #909399;
  font-size: 12px;
  line-height: 1.6;
}
.hint.inline {
  width: auto;
  margin-left: 12px;
}
</style>
