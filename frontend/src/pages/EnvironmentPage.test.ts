import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { listEnvironments } = vi.hoisted(() => ({
  listEnvironments: vi.fn(),
}))

vi.mock('@/api/environments', () => ({
  listEnvironments,
}))

import EnvironmentPage from './EnvironmentPage.vue'

describe('EnvironmentPage', () => {
  beforeEach(() => {
    listEnvironments.mockReset()
    listEnvironments.mockResolvedValue([
      {
        id: 'env-local-prod',
        name: '本地生产环境',
        code: 'local-prod',
        type: 'LOCAL',
        networkMode: 'DIRECT',
        status: 'HEALTHY',
        agentStatus: 'NOT_REQUIRED',
        lastCheckAt: '2026-06-07T12:40:00+08:00',
      },
      {
        id: 'env-project-x-prod',
        name: '项目 X 生产',
        code: 'project-x-prod',
        type: 'PROJECT',
        networkMode: 'AGENT',
        status: 'HEALTHY',
        agentStatus: 'ONLINE',
        lastCheckAt: '2026-06-07T12:40:12+08:00',
      },
      {
        id: 'env-project-z-prod',
        name: '项目 Z 生产',
        code: 'project-z-prod',
        type: 'PROJECT',
        networkMode: 'AGENT',
        status: 'UNKNOWN',
        agentStatus: 'OFFLINE',
        lastCheckAt: '2026-06-07T12:31:00+08:00',
      },
    ])
  })

  it('loads environments from API and shows V1 integration readiness from user view', async () => {
    const wrapper = mount(EnvironmentPage, {
      global: {
        stubs: {
          EnvironmentConfigDrawer: true,
          StatusTag: { template: '<span>{{ status }}</span>', props: ['status'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(listEnvironments).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('项目 X 生产')
    expect(wrapper.text()).toContain('项目环境')
    expect(wrapper.text()).toContain('Agent 模式')
    expect(wrapper.text()).toContain('ONLINE')
    expect(wrapper.text()).toContain('真实联调前需要准备 Agent Linux 主机')
    expect(wrapper.text()).toContain('1 个项目环境 Agent 未就绪')
  })

  it('filters environments by keyword and network mode', async () => {
    const wrapper = mount(EnvironmentPage, {
      global: {
        stubs: {
          EnvironmentConfigDrawer: true,
          StatusTag: { template: '<span>{{ status }}</span>', props: ['status'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    await wrapper.get('input').setValue('project-z')
    expect(wrapper.text()).toContain('项目 Z 生产')
    expect(wrapper.text()).not.toContain('项目 X 生产')
    expect(wrapper.text()).not.toContain('本地生产环境')

    await wrapper.get('.el-select').trigger('click')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Agent 模式')
  })

  it('shows a clear fallback when environment loading fails', async () => {
    listEnvironments.mockRejectedValueOnce(new Error('环境接口不可用'))

    const wrapper = mount(EnvironmentPage, {
      global: {
        stubs: {
          EnvironmentConfigDrawer: true,
          StatusTag: { template: '<span>{{ status }}</span>', props: ['status'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('环境接口不可用')
  })
})
