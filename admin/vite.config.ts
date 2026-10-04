import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'node:path'
import { writeHotFile } from './vite-plugins/write-hot-file.ts'

// 构建产物与 hot 文件输出到仓库根目录的 public/ 下, Go 服务以 /build 静态路由提供,
// internal/view/inertia 从 public/build/.vite/manifest.json 与 public/hot 解析资源地址
const publicDir = path.resolve(import.meta.dirname, '../public')

export default defineConfig({
    plugins: [vue(), writeHotFile(path.join(publicDir, 'hot'))],
    // 本项目不使用 Vite 的 public 目录拷贝功能 (避免与仓库根目录的 public/ 混淆)
    publicDir: false,
    build: {
        outDir: path.join(publicDir, 'build'),
        emptyOutDir: true,
        manifest: true,
        rollupOptions: {
            input: ['resources/js/app.ts', 'resources/css/app.css'],
        },
    },
    server: {
        host: '127.0.0.1',
        port: 5173,
        strictPort: true,
        // 页面由 Go 服务 (:3601) 提供, 开发模式下 import 的图片等资源需带上 Vite 地址, 否则会向 :3601 请求而 404
        origin: 'http://localhost:5173',
    },
})
