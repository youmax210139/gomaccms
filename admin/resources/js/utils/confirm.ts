import { ElMessageBox } from 'element-plus'

// confirmAction 弹出统一的确认框 (居中), 用户确认后返回 true, 取消或关闭返回 false.
// 用法: if (await confirmAction('确认删除该任务?')) { ... }
export const confirmAction = (message: string, title = '操作确认'): Promise<boolean> =>
    ElMessageBox.confirm(message, title, {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
        center: true,
    }).then(() => true, () => false)
