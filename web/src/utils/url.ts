/**
 * 拼接 MinIO 文件代理 URL
 * 后端通过 GET /api/files/*path 代理下载 MinIO 文件
 * 传入数据库存储的相对路径（如 avatar/{userID}/{fileID}.jpg），返回前端可访问的 URL
 * 路径为空/null 时返回空字符串，调用方可据此判断是否使用占位
 */
export function fileUrl(path: string | null | undefined): string {
  if (!path) return ''
  return `/api/files/${path}`
}
