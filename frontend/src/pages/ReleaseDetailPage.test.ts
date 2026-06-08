import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  getReleaseDetail,
  getAgentTaskStatus,
} = vi.hoisted(() => ({
  getReleaseDetail: vi.fn(),
  getAgentTaskStatus: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    params: {
      id: 'REL-20260607-031',
    },
    query: {},
    fullPath: '/releases/REL-20260607-031',
  }),
}))

vi.mock('@/api/releases', () => ({
  getReleaseDetail,
}))

vi.mock('@/api/agentTasks', () => ({
  getAgentTaskStatus,
}))

import ReleaseDetailPage from './ReleaseDetailPage.vue'

describe('ReleaseDetailPage', () => {
  beforeEach(() => {
    getReleaseDetail.mockReset()
    getAgentTaskStatus.mockReset()
  })

  it('shows static snapshot state when no agent task id is provided', async () => {
    getReleaseDetail.mockResolvedValue({
      id: 'REL-20260607-031',
      sourceBaselineId: 'BL-20260607-0001',
      targetEnvironmentName: '项目 X 生产',
      status: 'RUNNING',
      progress: 75,
      agentName: 'agent-project-x',
      steps: [
        { name: '构建产物校验', status: 'SUCCESS', message: '' },
        { name: '灰度发布', status: 'RUNNING', message: '' },
      ],
      failures: [],
      logs: ['[INFO] release detail snapshot'],
    })

    const wrapper = mount(ReleaseDetailPage, {
      global: {
        stubs: {
          DeployStepPanel: { template: '<div data-testid="step-panel" />', props: ['title', 'status', 'steps', 'activeStepName'] },
          LogTerminal: { template: '<div data-testid="log-terminal">{{ title }}|{{ badge }}</div>', props: ['title', 'logs', 'badge'] },
          MetricCard: { template: '<div class="metric-card">{{ label }}:{{ value }}|{{ foot }}|{{ tone }}</div>', props: ['label', 'value', 'foot', 'tone'] },
          ServiceFailureDrawer: { template: '<div data-testid="drawer" />', props: ['visible', 'failure'] },
          ElButton: { template: '<button :disabled="disabled"><slot /></button>', props: ['disabled', 'type', 'link'] },
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
          ElTable: { template: '<div><slot /></div>', props: ['data', 'class'] },
          ElTableColumn: { template: '<div><slot :row="{}" /></div>', props: ['prop', 'label', 'minWidth', 'fixed', 'width'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(getAgentTaskStatus).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('静态快照')
    expect(wrapper.text()).toContain('未关联实时 Agent 任务，展示详情快照')
    expect(wrapper.text()).toContain('离线回放')
  })
})
