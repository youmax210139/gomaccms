<template>
  <div class="login-page">
    <div class="login-side">
      <h1 class="welcome">欢迎使用{{ siteName || 'GoMacCMS' }}</h1>
      <!-- 等距插画: 本地 SVG, 不依赖外部图片 -->
      <svg class="illustration" viewBox="0 0 640 560" aria-hidden="true">
        <defs>
          <linearGradient id="lg-shield" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" stop-color="#2fbf7a" />
            <stop offset="1" stop-color="#138a55" />
          </linearGradient>
          <linearGradient id="lg-shield-in" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stop-color="#5fd69a" />
            <stop offset="1" stop-color="#1fa866" />
          </linearGradient>
          <linearGradient id="lg-beam" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stop-color="#3cc480" stop-opacity="0" />
            <stop offset="1" stop-color="#3cc480" stop-opacity=".22" />
          </linearGradient>
          <linearGradient id="lg-screen" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" stop-color="#4fd18f" />
            <stop offset="1" stop-color="#22a866" />
          </linearGradient>
        </defs>
        <!-- 底座 -->
        <g>
          <path d="M40 330 L210 250 L380 330 L210 410 Z" fill="#f1f3f7" />
          <path d="M40 330 L210 410 L210 470 L40 390 Z" fill="#dfe3ea" />
          <path d="M380 330 L210 410 L210 470 L380 390 Z" fill="#eceff4" />
          <path d="M70 330 L210 264 L350 330 L210 396 Z" fill="none" stroke="#3cc480" stroke-opacity=".35" stroke-width="1.5" />
          <path d="M100 330 L210 280 L320 330 L210 380 Z" fill="#e6f7ee" />
        </g>
        <!-- 光柱 -->
        <path d="M120 60 L300 60 L300 330 L120 330 Z" fill="url(#lg-beam)" />
        <!-- 盾牌 -->
        <g transform="translate(210 205)">
          <path d="M0 -150 C 45 -125 85 -118 105 -118 L 105 -20 C 105 50 55 95 0 120 C -55 95 -105 50 -105 -20 L -105 -118 C -85 -118 -45 -125 0 -150 Z" fill="url(#lg-shield)" />
          <path d="M0 -128 C 38 -107 70 -100 86 -100 L 86 -22 C 86 36 45 74 0 96 C -45 74 -86 36 -86 -22 L -86 -100 C -70 -100 -38 -107 0 -128 Z" fill="url(#lg-shield-in)" />
          <path d="M-46 -12 L -12 22 L 52 -48" fill="none" stroke="#fff" stroke-width="22" stroke-linecap="round" stroke-linejoin="round" />
        </g>
        <!-- 小方块 -->
        <g>
          <path d="M440 300 L470 286 L500 300 L470 314 Z" fill="#7fe0ad" />
          <path d="M440 300 L470 314 L470 344 L440 330 Z" fill="#3cc480" />
          <path d="M500 300 L470 314 L470 344 L500 330 Z" fill="#2aa86a" />
        </g>
        <!-- 电脑 -->
        <g transform="translate(470 150)">
          <path d="M-60 60 L60 0 L150 45 L30 105 Z" fill="#eef0f4" />
          <path d="M-60 60 L30 105 L30 120 L-60 75 Z" fill="#d9dde5" />
          <path d="M150 45 L30 105 L30 120 L150 60 Z" fill="#e6e9ef" />
          <path d="M-30 45 L-30 -50 L70 -100 L70 -5 Z" fill="#cfd4dd" />
          <path d="M-22 40 L-22 -45 L62 -88 L62 -3 Z" fill="url(#lg-screen)" />
          <rect x="0" y="-40" width="8" height="40" fill="#fff" opacity=".85" transform="skewY(-27)" />
          <rect x="14" y="-28" width="8" height="28" fill="#fff" opacity=".85" transform="skewY(-27)" />
          <rect x="28" y="-18" width="8" height="18" fill="#fff" opacity=".85" transform="skewY(-27)" />
        </g>
        <!-- 装饰柱 -->
        <g opacity=".5">
          <path d="M380 470 L420 450 L460 470 L460 560 L380 560 Z" fill="#cdeee0" />
          <path d="M540 420 L570 405 L600 420 L600 560 L540 560 Z" fill="#d7f2e6" />
        </g>
        <path d="M300 100 Q 400 60 470 110" fill="none" stroke="#3cc480" stroke-dasharray="4 6" stroke-opacity=".6" />
        <path d="M320 360 L430 300" fill="none" stroke="#3cc480" stroke-dasharray="4 6" stroke-opacity=".6" />
      </svg>
    </div>

    <div class="login-card">
      <h2 class="card-title">系统管理</h2>
      <form @submit.prevent="submit">
        <p v-if="form.errors.form" class="form-error">{{ form.errors.form }}</p>
        <label class="field-label">账号：</label>
        <el-input v-model="form.userName" size="large" :prefix-icon="User" placeholder="用户名 / 邮箱" autocomplete="username" />
        <label class="field-label">密码：</label>
        <el-input v-model="form.password" size="large" type="password" show-password :prefix-icon="Lock" placeholder="密码" autocomplete="current-password" />
        <template v-if="captcha">
          <label class="field-label">验证码：</label>
          <div class="captcha-row">
            <el-input v-model="form.captcha" size="large" :prefix-icon="CircleCheck" maxlength="4" placeholder="验证码" autocomplete="off" />
            <img v-if="captchaImage" :src="captchaImage" alt="验证码" title="看不清? 点击刷新" class="captcha-img" @click="refreshCaptcha">
          </div>
        </template>
        <el-button native-type="submit" class="login-btn" size="large" :loading="form.processing">立即登录</el-button>
      </form>
      <p class="copyright">© {{ siteName || 'GoMacCMS' }} All Rights Reserved.</p>
      <div class="disclaimer">
        <h4>免责声明</h4>
        <p>本程序仅提供影视资源的整理与展示, 不存储任何视频文件; 请在遵守当地法律的前提下使用, 对使用过程中产生的内容与后果, 本程序不承担任何责任。</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useForm } from '@inertiajs/vue3'
import { CircleCheck, Lock, User } from '@element-plus/icons-vue'
import { http } from '../utils/http'
import { resetTabs } from '../utils/tabs'

// captcha: 后台「登录验证码」是否开启; siteName: 默认站点名称
const props = defineProps<{ captcha?: boolean; siteName?: string }>()

const form = useForm({
    userName: '',
    password: '',
    captchaId: '',
    captcha: '',
})

// 验证码验证一次即失效, 每次提交后 (无论成功与否) 都换一张
const captchaImage = ref('')
const refreshCaptcha = async () => {
    if (!props.captcha) return
    const resp = await http.get('/login/captcha')
    if (resp.data.code === 0) {
        form.captchaId = resp.data.data.id
        captchaImage.value = resp.data.data.image
    }
    form.captcha = ''
}
onMounted(() => {
    resetTabs()
    refreshCaptcha()
})

const submit = () => {
    form.post('/login', { onFinish: refreshCaptcha })
}
</script>

<style scoped>
.login-page {
  --green: #3cc480;
  position: relative;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 40px;
  min-height: 100vh;
  padding: 40px 6vw;
  box-sizing: border-box;
  overflow: hidden;
  background:
    radial-gradient(ellipse at 20% 40%, rgba(60, 196, 128, .10), transparent 55%),
    linear-gradient(160deg, #f6f7fa 0%, #eef1f6 60%, #f5f6f9 100%);
}
.login-side {
  flex: 1;
  max-width: 720px;
  align-self: flex-start;
}
.welcome {
  margin: 20px 0 0 60px;
  font-size: 54px;
  font-weight: 700;
  letter-spacing: 2px;
  color: var(--green);
  white-space: nowrap;
}
.illustration {
  display: block;
  width: 100%;
  max-width: 680px;
  margin-top: -10px;
}
.login-card {
  flex-shrink: 0;
  width: 420px;
  padding: 32px 30px 26px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 10px 40px rgba(30, 40, 60, .08);
}
.card-title {
  margin: 0 0 18px;
  font-size: 22px;
  font-weight: 700;
  color: #222;
}
.field-label {
  display: block;
  margin: 22px 0 10px;
  font-size: 15px;
  color: #333;
}
.form-error {
  margin: 0 0 4px;
  padding: 8px 12px;
  border-radius: 4px;
  background: #fef0f0;
  color: #f56c6c;
  font-size: 13px;
}
.captcha-row {
  display: flex;
  gap: 16px;
  align-items: center;
}
.captcha-row .el-input {
  flex: 1;
}
.captcha-img {
  width: 140px;
  height: 40px;
  border-radius: 4px;
  background: #f0f8ff;
  cursor: pointer;
}
.login-btn {
  width: 100%;
  margin-top: 28px;
  border: none;
  background: var(--green);
  color: #fff;
}
.login-btn:hover,
.login-btn:focus {
  background: #34b373;
  color: #fff;
}
.copyright {
  margin: 18px 0 0;
  text-align: center;
  font-size: 13px;
  color: #aaa;
}
.disclaimer h4 {
  margin: 18px 0 8px;
  font-size: 15px;
  font-weight: 500;
  color: #333;
}
.disclaimer p {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: #444;
}
.login-page :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--green) inset;
}
@media (max-width: 960px) {
  .login-page {
    justify-content: center;
    padding: 24px 16px;
  }
  .login-side {
    display: none;
  }
  .login-card {
    width: 100%;
    max-width: 420px;
  }
}
</style>
