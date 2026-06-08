import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  getDeployTaskDetail,
  getAgentTaskStatus,
  messageWarning,
} = vi.hoisted(() => ({
  getDeployTaskDetail: vi.fn(),
  getAgentTaskStatus: vi.fn(),
  messageWarning: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    params: {
      id: 'DEP-20260607-009',
    },
    query: {},
    fullPath: '/deploy-tasks/DEP-20260607-009',
  }),
}))

vi.mock('element-plus', () => ({
  ElButton: { template: '<button />' },
  ElCard: { template: '<div><slot name="header" /><slot /></div>' },
  ElLoadingDirective: {
    mounted: () => undefined,
    updated: () => undefined,
    unmounted: () => undefined,
  },
  ElMessage: {
    warning: messageWarning,
  },
}))

vi.mock('@/api/deployTasks', () => ({
  getDeployTaskDetail,
}))

vi.mock('@/api/agentTasks', () => ({
  getAgentTaskStatus,
}))

import DeployDetailPage from './DeployDetailPage.vue'

describe('DeployDetailPage', () => {
  beforeEach(() => {
    getDeployTaskDetail.mockReset()
    getAgentTaskStatus.mockReset()
    messageWarning.mockReset()
  })

  it('falls back to mock detail and warns when request fails', async () => {
    getDeployTaskDetail.mockRejectedValue(new Error('network error'))

    const wrapper = mount(DeployDetailPage, {
      global: {
        stubs: {
          DeployStepPanel: { template: '<div data-testid="step-panel" />', props: ['title', 'status', 'steps', 'activeStepName'] },
          LogTerminal: { template: '<div data-testid="log-terminal">{{ title }}|{{ badge }}</div>', props: ['title', 'logs', 'badge'] },
          MetricCard: { template: '<div class="metric-card">{{ label }}:{{ value }}|{{ foot }}|{{ tone }}</div>', props: ['label', 'value', 'foot', 'tone'] },
          ElButton: { template: '<button :disabled="disabled"><slot /></button>', props: ['disabled', 'type', 'link'] },
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(messageWarning).toHaveBeenCalledWith('加载部署任务详情失败，已显示本地示例数据')
    expect(getAgentTaskStatus).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('部署任务详情：DEP-20260607-009')
    expect(wrapper.text()).toContain('静态快照')
  })
})
