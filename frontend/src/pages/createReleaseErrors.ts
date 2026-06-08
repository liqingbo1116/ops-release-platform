import { ApiClientError } from '@/api/client'

export function resolveCreateReleaseErrorMessage(
  error: unknown,
  releaseMode: 'SERVICE_RELEASE' | 'SERVICE_DEPLOYMENT',
): string {
  if (error instanceof ApiClientError) {
    if (error.message === 'agent not found') {
      return '所选 Agent 不存在，请重新选择'
    }
    if (error.message === 'agent must be ONLINE') {
      return '所选 Agent 当前离线，请切换为在线 Agent'
    }
    if (error.message === 'agent does not belong to target environment') {
      return '所选 Agent 与目标环境不匹配，请重新选择'
    }
  }

  return releaseMode === 'SERVICE_DEPLOYMENT' ? '创建服务部署任务失败' : '提交服务发版失败'
}
