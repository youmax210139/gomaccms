const pad = (n: number) => String(n).padStart(2, '0')

// formatDateTime 固定为 YYYY-MM-DD HH:mm:ss (24 小时制, 零填充, 与后端 time.DateTime 一致), 不随浏览器语系变化;
// 值为空时返回 empty
export const formatDateTime = (value: string | number | Date | null | undefined, empty = '') => {
    if (!value) return empty
    const d = new Date(value)
    if (Number.isNaN(d.getTime())) return empty
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

// formatStamp Unix 时间戳 (秒) 的 formatDateTime
export const formatStamp = (seconds: number | null | undefined, empty = '') => formatDateTime(seconds ? seconds * 1000 : null, empty)
