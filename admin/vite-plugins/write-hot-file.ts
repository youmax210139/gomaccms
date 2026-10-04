import type { Plugin, ViteDevServer } from 'vite'
import { writeFileSync, rmSync, existsSync } from 'node:fs'

export function writeHotFile(hotFilePath: string): Plugin {
    return {
        name: 'write-hot-file',
        configureServer(server: ViteDevServer) {
            const cleanup = () => {
                if (existsSync(hotFilePath)) rmSync(hotFilePath)
            }
            process.on('exit', cleanup)
            process.on('SIGINT', () => {
                cleanup()
                process.exit()
            })
            server.httpServer?.once('listening', () => {
                const address = server.httpServer?.address()
                const port = typeof address === 'object' && address ? address.port : 5173
                const protocol = server.config.server.https ? 'https' : 'http'
                writeFileSync(hotFilePath, `${protocol}://localhost:${port}`)
            })
        },
    }
}
