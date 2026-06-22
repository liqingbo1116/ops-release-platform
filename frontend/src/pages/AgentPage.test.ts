import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { claimAgent, listAgents, listEnvironments } = vi.hoisted(() => ({
  claimAgent: vi.fn(),
  listAgents: vi.fn(),
  listEnvironments: vi.fn(),
}))

vi.mock('@/api/agents', () => ({
  claimAgent,
  listAgents,
}))

vi.mock('@/api/environments', () => ({
  listEnvironments,
}))

import AgentPage from './AgentPage.vue'

describe('AgentPage', () => {
  beforeEach(() => {
    listAgents.mockReset()
    listEnvironments.mockReset()
    claimAgent.mockReset()
    listAgents.mockResolvedValue([
      {
        id: 'agent-project-x',
        name: 'agent-project-x',
        environmentId: 'env-project-x-prod',
        environmentName: '项目 X 生产',
        version: '1.3.2',
        status: 'ONLINE',
        claimStatus: 'CLAIMED',
        capabilities: ['remote-probe', 'k8s-api', 'http-check'],
        runtimeStatus: {
          kubernetes: {
            status: 'HEALTHY',
            message: '远程 Kubernetes 连接正常',
            updatedAt: '2026-06-07T12:40:12+08:00',
            items: ['xjzt-prod', 'xjzt-job'],
          },
          harbor: {
            status: 'HEALTHY',
            message: '远程 Harbor 连接正常',
            updatedAt: '2026-06-07T12:40:12+08:00',
            items: ['xjzt'],
          },
        },
        lastHeartbeatAt: '2026-06-07T12:40:12+08:00',
        currentTaskId: 'REL-20260607-031',
      },
      {
        id: 'agent-project-z',
        name: 'agent-project-z',
        environmentId: 'env-project-z-prod',
        environmentName: '项目 Z 生产',
        version: '1.2.8',
        status: 'OFFLINE',
        claimStatus: 'CLAIMED',
        capabilities: ['http-check'],
        lastHeartbeatAt: '2026-06-07T12:31:00+08:00',
        currentTaskId: null,
      },
      {
        id: 'agent-project-new',
        name: 'agent-project-new',
        environmentId: '',
        environmentName: '',
        version: '1.0.0',
        status: 'ONLINE',
        claimStatus: 'PENDING_CLAIM',
        capabilities: ['k8s-api'],
        lastHeartbeatAt: '2026-06-07T12:42:00+08:00',
        currentTaskId: null,
      },
    ])
    listEnvironments.mockResolvedValue([
      {
        id: 'env-local-prod',
        name: '本地生产',
        code: 'local-prod',
        type: 'LOCAL',
        networkMode: 'DIRECT',
        clusterId: 'local',
        registryId: 'local',
        status: 'HEALTHY',
        agentStatus: '',
        lastCheckAt: '2026-06-07T12:40:12+08:00',
      },
      {
        id: 'env-project-x-prod',
        name: '项目 X 生产',
        code: 'project-x-prod',
        type: 'PROJECT',
        networkMode: 'AGENT',
        clusterId: 'remote',
        registryId: 'remote',
        status: 'HEALTHY',
        agentStatus: 'ONLINE',
        lastCheckAt: '2026-06-07T12:40:12+08:00',
      },
      {
        id: 'env-project-new-prod',
        name: '项目 New 生产',
        code: 'project-new-prod',
        type: 'PROJECT',
        networkMode: 'AGENT',
        clusterId: 'remote',
        registryId: 'remote',
        status: 'PENDING',
        agentStatus: 'ONLINE',
        lastCheckAt: null,
      },
    ])
  })

  it('shows Agent readiness, heartbeat, capabilities, recent task, and offline blocker', async () => {
    const wrapper = mount(AgentPage, {
      global: {
        stubs: {
          AgentRegisterDrawer: true,
          MetricCard: { template: '<div>{{ label }} {{ value }} {{ foot }}</div>', props: ['label', 'value', 'foot'] },
          StatusTag: { template: '<span>{{ status }}</span>', props: ['status'] },
          ElTooltip: { template: '<div class="tooltip-stub" :data-content="content"><slot /></div>', props: ['content'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(listAgents).toHaveBeenCalledTimes(1)
    expect(listEnvironments).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('二进制直接启动')
    expect(wrapper.text()).toContain('agent-project-x')
    expect(wrapper.text()).toContain('项目 X 生产')
    expect(wrapper.text()).toContain('remote-probe / k8s-api / http-check')
    expect(wrapper.text()).toContain('K8s')
    expect(wrapper.text()).toContain('Harbor')
    expect(wrapper.text()).toContain('HEALTHY')
    expect(wrapper.findAll('.tooltip-stub').map((item) => item.attributes('data-content'))).toContain(
      '远程 Kubernetes 连接正常，已上报 2 个 namespace',
    )
    expect(wrapper.findAll('.tooltip-stub').map((item) => item.attributes('data-content'))).toContain(
      '远程 Harbor 连接正常，已上报 1 个 project',
    )
    expect(wrapper.text()).toContain('待绑定')
    expect(wrapper.text()).not.toContain('远程探测')
    expect(wrapper.text()).toContain('1 个 Agent 离线')
  })

  it('filters agents by environment and capability', async () => {
    const wrapper = mount(AgentPage, {
      global: {
        stubs: {
          AgentRegisterDrawer: true,
          MetricCard: true,
          StatusTag: { template: '<span>{{ status }}</span>', props: ['status'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    await wrapper.get('input').setValue('project-z')
    expect(wrapper.text()).toContain('agent-project-z')
    expect(wrapper.text()).not.toContain('agent-project-x')

    await wrapper.get('input').setValue('k8s-api')
    expect(wrapper.text()).toContain('agent-project-x')
    expect(wrapper.text()).not.toContain('agent-project-z')
  })

  it('binds a pending Agent to a remote product only', async () => {
    claimAgent.mockResolvedValue({
      id: 'agent-project-new',
      name: 'agent-project-new',
      environmentId: 'env-project-new-prod',
      environmentName: '项目 New 生产',
      version: '1.0.0',
      status: 'ONLINE',
      claimStatus: 'CLAIMED',
      capabilities: ['k8s-api'],
      lastHeartbeatAt: '2026-06-07T12:42:00+08:00',
      currentTaskId: null,
    })
    const wrapper = mount(AgentPage, {
      global: {
        stubs: {
          AgentRegisterDrawer: true,
          MetricCard: true,
          StatusTag: { template: '<span>{{ status }}</span>', props: ['status'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()
    await wrapper.findAll('button').find((button) => button.text() === '绑定产品')?.trigger('click')
    await flushPromises()

    const optionLabels = wrapper.findAllComponents({ name: 'ElOption' }).map((option) => option.props('label'))
    expect(optionLabels).toContain('项目 New 生产')
    expect(optionLabels).not.toContain('本地生产')

    await wrapper.findComponent({ name: 'ElSelect' }).vm.$emit('update:modelValue', 'env-project-new-prod')
    await flushPromises()
    await wrapper.findAll('button').find((button) => button.text() === '确认绑定')?.trigger('click')
    await flushPromises()

    expect(claimAgent).toHaveBeenCalledWith('agent-project-new', 'env-project-new-prod')
  })
})
