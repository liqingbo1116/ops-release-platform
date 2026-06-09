import { describe, expect, it } from 'vitest'

import { ApiClientError } from '@/api/client'

import { resolveCreateReleaseErrorMessage } from './createReleaseErrors'

describe('resolveCreateReleaseErrorMessage', () => {
  it('maps missing agent validation to a specific release message', () => {
    const message = resolveCreateReleaseErrorMessage(
      new ApiClientError('agent not found', { status: 400, code: 'VALIDATION_ERROR' }),
      'SERVICE_RELEASE',
    )

    expect(message).toBe('所选 Agent 不存在，请重新选择')
  })

  it('maps offline agent validation to a specific release message', () => {
    const message = resolveCreateReleaseErrorMessage(
      new ApiClientError('agent must be ONLINE', { status: 400, code: 'VALIDATION_ERROR' }),
      'SERVICE_RELEASE',
    )

    expect(message).toBe('所选 Agent 当前离线，请切换为在线 Agent')
  })

  it('maps environment mismatch validation to a specific deployment message', () => {
    const message = resolveCreateReleaseErrorMessage(
      new ApiClientError('agent does not belong to target environment', { status: 400, code: 'VALIDATION_ERROR' }),
      'SERVICE_DEPLOYMENT',
    )

    expect(message).toBe('所选 Agent 与目标环境不匹配，请重新选择')
  })

  it('maps Jenkins readiness failure to an environment preparation message', () => {
    const message = resolveCreateReleaseErrorMessage(
      new ApiClientError('jenkins trigger failed', { status: 400, code: 'VALIDATION_ERROR' }),
      'SERVICE_RELEASE',
    )

    expect(message).toBe('Jenkins 任务触发失败，请确认 Jenkins 环境和 Job 已准备')
  })

  it('maps Harbor image readiness failure to an environment preparation message', () => {
    const message = resolveCreateReleaseErrorMessage(
      new ApiClientError('registry image check failed', { status: 400, code: 'VALIDATION_ERROR' }),
      'SERVICE_RELEASE',
    )

    expect(message).toBe('Harbor 镜像查询失败，请确认镜像仓库环境已准备')
  })

  it('maps Kubernetes readiness failure to an environment preparation message', () => {
    const message = resolveCreateReleaseErrorMessage(
      new ApiClientError('kubernetes workload probe failed', { status: 400, code: 'VALIDATION_ERROR' }),
      'SERVICE_DEPLOYMENT',
    )

    expect(message).toBe('Kubernetes 工作负载探测失败，请确认目标集群和 namespace 已准备')
  })

  it('maps permission failure by status code', () => {
    const message = resolveCreateReleaseErrorMessage(
      new ApiClientError('forbidden', { status: 403, code: 'FORBIDDEN' }),
      'SERVICE_DEPLOYMENT',
    )

    expect(message).toBe('当前账号没有服务部署权限')
  })

  it('falls back to a generic release error', () => {
    expect(resolveCreateReleaseErrorMessage(new Error('boom'), 'SERVICE_RELEASE')).toBe('提交服务发版失败')
  })

  it('falls back to a generic deployment error', () => {
    expect(resolveCreateReleaseErrorMessage(new Error('boom'), 'SERVICE_DEPLOYMENT')).toBe('创建服务部署任务失败')
  })
})
