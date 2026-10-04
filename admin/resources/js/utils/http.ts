import axios from 'axios'

// 同源 + cookie 自动带认证,不需要手动附加任何 auth header。
const http = axios.create({
    baseURL: '/',
    timeout: 30000,
})

export { http }
