import { computed, type Component } from 'vue'
import { usePage } from '@inertiajs/vue3'
import { FolderOpened, Menu, MagicStick, Timer, Document, VideoPlay, User } from '@element-plus/icons-vue'

// perm: 后台权限 key (与 internal/http/middleware/permission.go 一致), 没有该权限的管理员不显示该菜单
export type MenuItem = { path: string; title: string; perm?: string }
export type MenuGroup = { key: string; title: string; icon: Component; items: MenuItem[] }

// 首页 (固定页签, 不可关闭)
export const HOME: MenuItem = { path: '/manage/index', title: '首页' }

// 侧边栏菜单; 页签标题也从这里按路径查找
export const menuGroups: MenuGroup[] = [
    {
        key: 'site', title: '系统', icon: Menu, items: [
            { path: '/manage/config/basic', title: '站群管理', perm: 'config' },
            { path: '/manage/banner/list', title: '海报管理', perm: 'banner' },
            { path: '/manage/publish/center', title: '网站地图', perm: 'publish' },
            { path: '/manage/cache/list', title: '缓存管理', perm: 'cache' },
        ],
    },
    {
        key: 'film', title: '基礎', icon: Document, items: [
            { path: '/manage/film/class/tree', title: '分类管理', perm: 'category' },
            { path: '/manage/ad/list', title: '广告管理', perm: 'ad' },
            { path: '/manage/lang/list', title: '语言管理', perm: 'lang' },
        ],
    },
    {
        key: 'vod', title: '视频', icon: VideoPlay, items: [
            { path: '/manage/film/player', title: '播放器', perm: 'player' },
            // 添加视频在「视频数据」页的「添加」按钮中, 不另设菜单
            { path: '/manage/film/search/list', title: '视频数据', perm: 'vod' },
        ],
    },
    {
        key: 'collect', title: '采集管理', icon: MagicStick, items: [
            { path: '/manage/collect/list', title: '采集接口', perm: 'collect' },
        ],
    },
    {
        key: 'cron', title: '定时任务', icon: Timer, items: [
            { path: '/manage/cron/list', title: '任务管理', perm: 'cron' },
        ],
    },
    {
        key: 'file', title: '文件管理', icon: FolderOpened, items: [
            { path: '/manage/file/upload', title: '图库管理', perm: 'gallery' },
        ],
    },
    {
        key: 'user', title: '用户', icon: User, items: [
            { path: '/manage/admin/list', title: '管理员', perm: 'admin' },
            { path: '/manage/member/group/list', title: '会员组', perm: 'member.group' },
            { path: '/manage/member/list', title: '会员', perm: 'member' },
            { path: '/manage/admin/log/list', title: '操作日志', perm: 'admin.log' },
        ],
    },
]

export type Access = { founder?: boolean; permissions?: string[] | null }

// can 当前管理员是否有某个权限 (创始管理员拥有全部权限)
export const can = (user: Access | undefined, perm?: string) =>
    !perm || !!user?.founder || !!user?.permissions?.includes(perm)

// useCan 页面中按当前管理员的权限显示操作按钮: const can = useCan(); v-if="can('vod.delete')"
export const useCan = () => {
    const page = usePage<{ currentUser?: Access }>()
    const user = computed(() => page.props.currentUser)
    return (perm?: string) => can(user.value, perm)
}

// visibleGroups 按权限过滤后的侧边栏菜单 (没有可见菜单项的分组不显示)
export const visibleGroups = (user: Access | undefined) => menuGroups
    .map((g) => ({ ...g, items: g.items.filter((i) => can(user, i.perm)) }))
    .filter((g) => g.items.length > 0)

const titles = new Map<string, string>([
    [HOME.path, HOME.title],
    ...menuGroups.flatMap((g) => g.items.map((i) => [i.path, i.title] as [string, string])),
])

// titleOf 返回菜单中该路径的标题; 不在菜单中的页面返回 undefined (不产生页签)
export const titleOf = (path: string) => titles.get(path)

// pathOf 从 Inertia 的 page.url (可能带查询参数) 取出路径部分
export const pathOf = (url: string) => new URL(url, window.location.origin).pathname
