import { createApp, h, reactive, type DefineComponent } from 'vue'
import { createInertiaApp, router } from '@inertiajs/vue3'
import ElementPlus, { ElConfigProvider, ElMessage } from 'element-plus'

// ElMessage 提示垂直置中: Element Plus 只能设置距顶部的 offset, 按视窗高度计算 (提示框高约 40px),
// 多条提示会从中间依次往下排列; 相同内容的提示合并显示
const MESSAGE_HEIGHT = 40
const messageConfig = reactive({ offset: 16, grouping: true })
const centerMessages = () => {
    messageConfig.offset = Math.max(16, Math.round((window.innerHeight - MESSAGE_HEIGHT) / 2))
}
centerMessages()
window.addEventListener('resize', centerMessages)

createInertiaApp({
    resolve: (name) => {
        const pages = import.meta.glob<DefineComponent>('./Pages/**/*.vue', { eager: true })
        const page = pages[`./Pages/${name}.vue`]
        if (!page) {
            throw new Error(`Page not found: ./Pages/${name}.vue`)
        }
        return page
    },
    setup({ el, App, props, plugin }) {
        createApp({ render: () => h(ElConfigProvider, { message: messageConfig }, () => h(App, props)) })
            .use(plugin)
            .use(ElementPlus)
            .mount(el)
    },
})

// 后台操作结果的统一提示: /manage/* 下的提交 (POST 保存/新增/更新/开关) 与 GET 删除/重置,
// 后端成功时 redirect 回页面, 失败时带 errors.form 重新渲染页面.
// 普通页面浏览 (GET 列表/分页/筛选) 不提示; 登录等非 /manage 请求不提示.
let lastVisit: { method: string; path: string } | null = null
router.on('start', (event) => {
    const { method, url } = event.detail.visit
    lastVisit = { method, path: url.pathname }
})
router.on('success', (event) => {
    const visit = lastVisit
    lastVisit = null
    if (!visit || !visit.path.startsWith('/manage/')) return
    const error = (event.detail.page.props.errors as { form?: string } | undefined)?.form
    const mutation = visit.method !== 'get' || visit.path.endsWith('/del') || visit.path.endsWith('/reset')
    if (!mutation) return
    if (error) {
        ElMessage.error(error)
    } else if (visit.path.endsWith('/del')) {
        ElMessage.success('删除成功')
    } else if (visit.path.endsWith('/reset')) {
        ElMessage.success('已重置')
    } else {
        ElMessage.success('保存成功')
    }
})
