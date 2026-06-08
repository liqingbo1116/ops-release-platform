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

  it('falls back to a generic release error', () => {
    expect(resolveCreateReleaseErrorMessage(new Error('boom'), 'SERVICE_RELEASE')).toBe('提交服务发版失败')
  })

  it('falls back to a generic deployment error', () => {
    expect(resolveCreateReleaseErrorMessage(new Error('boom'), 'SERVICE_DEPLOYMENT')).toBe('创建服务部署任务失败')
  })
})
