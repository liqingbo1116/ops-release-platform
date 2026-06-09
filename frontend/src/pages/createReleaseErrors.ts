import { ApiClientError } from '@/api/client'

export function resolveCreateReleaseErrorMessage(
  error: unknown,
  releaseMode: 'SERVICE_RELEASE' | 'SERVICE_DEPLOYMENT',
): string {
  if (error instanceof ApiClientError) {
    if (error.message === 'baseline not found') {
      return '来源基线不存在，请重新选择后再试'
    }
    if (error.message === 'agent not found') {
      return '所选 Agent 不存在，请重新选择'
    }
    if (error.message === 'agent must be ONLINE') {
      return '所选 Agent 当前离线，请切换为在线 Agent'
    }
    if (error.message === 'agent does not belong to target environment') {
      return '所选 Agent 与目标环境不匹配，请重新选择'
    }
    if (error.message === 'release services must come from NEED_UPDATE diff items') {
      return '发布单只能提交差异结果中的需更新服务'
    }
    if (error.message === 'service release must not include source baseline') {
      return '服务发版不需要来源基线，请从目标已有服务发起发版'
    }
    if (error.message === 'source baseline is required for service deployment') {
      return '服务部署必须先选择来源基线并完成差异对比'
    }
    if (error.message === 'deploy services must come from MISSING_IN_TARGET diff items') {
      return '部署任务只能提交目标环境缺失的服务'
    }
    if (error.message === 'jenkins trigger failed') {
      return 'Jenkins 任务触发失败，请确认 Jenkins 环境和 Job 已准备'
    }
    if (error.message === 'registry image check failed') {
      return 'Harbor 镜像查询失败，请确认镜像仓库环境已准备'
    }
    if (error.message === 'registry image sync failed') {
      return 'Harbor 镜像同步失败，请确认 Agent 到镜像仓库的网络和凭证'
    }
    if (error.message === 'release image not found') {
      return '所选 Harbor 镜像不存在，请重新选择镜像 tag'
    }
    if (error.message === 'kubernetes workload probe failed') {
      return 'Kubernetes 工作负载探测失败，请确认目标集群和 namespace 已准备'
    }
    if (error.status === 403 || error.message === 'permission denied' || error.message === 'environment permission denied') {
      return releaseMode === 'SERVICE_DEPLOYMENT' ? '当前账号没有服务部署权限' : '当前账号没有服务发版权限'
    }
  }

  return releaseMode === 'SERVICE_DEPLOYMENT' ? '创建服务部署任务失败' : '提交服务发版失败'
}
