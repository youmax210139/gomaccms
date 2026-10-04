<template>
  <div class="profile">
    <p v-if="page.props.errors?.form" class="form-error">{{ page.props.errors.form }}</p>
    <el-divider content-position="left">个人资料</el-divider>
    <el-form :model="info" label-width="90px">
      <el-form-item label="账号">
        <span>{{ user.userName }}</span>
        <el-tag v-if="user.founder" size="small" type="danger" class="tag">创始人</el-tag>
      </el-form-item>
      <el-form-item label="头像">
        <div class="avatar-row">
          <el-avatar v-if="info.avatar" :size="56" :src="info.avatar"/>
          <el-avatar v-else :size="56" :icon="UserFilled" class="default-avatar"/>
          <el-button @click="pickerV = true">从图库选择</el-button>
          <el-button v-if="info.avatar" link @click="info.avatar = ''">使用默认头像</el-button>
        </div>
      </el-form-item>
      <el-form-item label="昵称" required>
        <el-input v-model="info.nickName" maxlength="60"/>
      </el-form-item>
      <el-form-item label="Email">
        <el-input v-model="info.email" maxlength="100"/>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="info.processing" @click="saveInfo">保存资料</el-button>
      </el-form-item>
    </el-form>

    <el-divider content-position="left">修改密码</el-divider>
    <el-form :model="pwd" label-width="90px">
      <el-form-item label="原密码" required>
        <el-input v-model="pwd.password" type="password" show-password autocomplete="current-password"/>
      </el-form-item>
      <el-form-item label="新密码" required>
        <el-input v-model="pwd.newPassword" type="password" show-password autocomplete="new-password" placeholder="至少 6 个字符"/>
      </el-form-item>
      <el-form-item label="确认密码" required>
        <el-input v-model="confirmPwd" type="password" show-password autocomplete="new-password"/>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="pwd.processing" @click="savePwd">修改密码</el-button>
      </el-form-item>
    </el-form>

    <GalleryPicker v-model:visible="pickerV" :model-value="info.avatar" @select="(link: string) => { info.avatar = link; pickerV = false }"/>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useForm, usePage } from '@inertiajs/vue3'
import { ElMessage } from 'element-plus'
import { UserFilled } from '@element-plus/icons-vue'
import AdminLayout from '../../Layouts/AdminLayout.vue'
import GalleryPicker from '../../Components/GalleryPicker.vue'

type CurrentUser = { userName: string; nickName: string; email: string; avatar: string; founder: boolean }

const page = usePage<{ errors?: { form?: string }; currentUser: CurrentUser }>()

defineOptions({ layout: AdminLayout })

const user = computed(() => page.props.currentUser)
const avatarOf = (a: string) => (a && a !== 'empty' ? a : '')

const info = useForm({ nickName: user.value.nickName, email: user.value.email, avatar: avatarOf(user.value.avatar) })
const pickerV = ref(false)
const saveInfo = () => info.post('/manage/user/profile', {
    preserveScroll: true,
    onSuccess: () => { if (!page.props.errors?.form) ElMessage.success('资料已保存') },
})

const pwd = useForm({ password: '', newPassword: '' })
const confirmPwd = ref('')
const savePwd = () => {
    if (pwd.newPassword !== confirmPwd.value) {
        ElMessage.warning('两次输入的新密码不一致')
        return
    }
    pwd.post('/manage/user/password', {
        preserveScroll: true,
        onSuccess: () => {
            if (page.props.errors?.form) return
            pwd.reset()
            confirmPwd.value = ''
            ElMessage.success('密码已修改')
        },
    })
}
</script>

<style scoped>
.profile {
  max-width: 640px;
}
.tag {
  margin-left: 8px;
}
.avatar-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.default-avatar {
  background: var(--el-color-info-light-5);
  color: #fff;
}
</style>
