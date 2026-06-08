import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  getReleaseDetail,
  getAgentTaskStatus,
  retryRelease,
  rollbackRelease,
  messageSuccess,
  messageWarning,
} = vi.hoisted(() => ({
  getReleaseDetail: vi.fn(),
  getAgentTaskStatus: vi.fn(),
  retryRelease: vi.fn(),
  rollbackRelease: vi.fn(),
  messageSuccess: vi.fn(),
  messageWarning: vi.fn(),
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

vi.mock('element-plus', async (importOriginal) => {
  const actual = await importOriginal<typeof import('element-plus')>()
  return {
    ...actual,
    ElMessage: {
      ...actual.ElMessage,
      success: messageSuccess,
      warning: messageWarning,
    },
  }
})

vi.mock('@/api/releases', () => ({
  getReleaseDetail,
  retryRelease,
  rollbackRelease,
}))

vi.mock('@/api/agentTasks', () => ({
  getAgentTaskStatus,
}))

import ReleaseDetailPage from './ReleaseDetailPage.vue'

describe('ReleaseDetailPage', () => {
  beforeEach(() => {
    getReleaseDetail.mockReset()
    getAgentTaskStatus.mockReset()
    retryRelease.mockReset()
    rollbackRelease.mockReset()
    messageSuccess.mockReset()
    messageWarning.mockReset()
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
      actionRecords: [{ action: 'CREATE_RELEASE', operator: 'li.si', status: 'SUCCESS', message: 'created', occurredAt: '2026-06-08T10:00:00Z' }],
      report: null,
      logs: ['[INFO] release detail snapshot'],
    })

    const wrapper = mount(ReleaseDetailPage, {
      global: {
        stubs: {
          DeployStepPanel: { template: '<div data-testid="step-panel" />', props: ['title', 'status', 'steps', 'activeStepName'] },
          LogTerminal: { template: '<div data-testid="log-terminal">{{ title }}|{{ badge }}</div>', props: ['title', 'logs', 'badge'] },
          MetricCard: { template: '<div class="metric-card">{{ label }}:{{ value }}|{{ foot }}|{{ tone }}</div>', props: ['label', 'value', 'foot', 'tone'] },
          ServiceFailureDrawer: { template: '<div data-testid="drawer" />', props: ['visible', 'failure'] },
          StatusTag: { template: '<span class="status-tag">{{ status }}</span>', props: ['status'] },
          ElButton: { template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>', props: ['disabled', 'type', 'link', 'loading'] },
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
          ElDialog: { template: '<div><slot /></div>', props: ['modelValue', 'title', 'width'] },
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

  it('polls agent status from detail agent task id and shows waiting confirm state', async () => {
    getReleaseDetail.mockResolvedValue({
      id: 'REL-20260607-031',
      sourceBaselineId: 'BL-20260607-0001',
      targetEnvironmentName: '项目 X 生产',
      status: 'RUNNING',
      progress: 75,
      agentName: 'agent-project-x',
      agentTaskId: 'AGT-REL-031',
      steps: [
        { name: '构建产物校验', status: 'SUCCESS', message: '' },
        { name: '灰度发布', status: 'RUNNING', message: '' },
      ],
      failures: [],
      actionRecords: [{ action: 'WAIT_CONFIRM', operator: 'system', status: 'PENDING_CONFIRM', message: 'waiting', occurredAt: '2026-06-08T10:00:00Z' }],
      report: null,
      logs: ['[INFO] release detail snapshot'],
    })
    getAgentTaskStatus.mockResolvedValue({
      enabled: true,
      status: {
        taskId: 'AGT-REL-031',
        type: 'release',
        step: '灰度发布',
        status: 'WAITING_CONFIRM',
        updatedAt: '2026-06-08T10:00:00Z',
      },
      logs: ['[WARN] waiting confirm'],
    })

    const wrapper = mount(ReleaseDetailPage, {
      global: {
        stubs: {
          DeployStepPanel: { template: '<div data-testid="step-panel" />', props: ['title', 'status', 'steps', 'activeStepName'] },
          LogTerminal: { template: '<div data-testid="log-terminal">{{ title }}|{{ badge }}</div>', props: ['title', 'logs', 'badge'] },
          MetricCard: { template: '<div class="metric-card">{{ label }}:{{ value }}|{{ foot }}|{{ tone }}</div>', props: ['label', 'value', 'foot', 'tone'] },
          ServiceFailureDrawer: { template: '<div data-testid="drawer" />', props: ['visible', 'failure'] },
          StatusTag: { template: '<span class="status-tag">{{ status }}</span>', props: ['status'] },
          ElButton: { template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>', props: ['disabled', 'type', 'link', 'loading'] },
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
          ElDialog: { template: '<div><slot /></div>', props: ['modelValue', 'title', 'width'] },
          ElTable: { template: '<div><slot /></div>', props: ['data', 'class'] },
          ElTableColumn: { template: '<div><slot :row="{}" /></div>', props: ['prop', 'label', 'minWidth', 'fixed', 'width'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(getAgentTaskStatus).toHaveBeenCalledWith('AGT-REL-031')
    expect(wrapper.text()).toContain('实时 Agent')
    expect(wrapper.text()).toContain('待确认')
    expect(wrapper.text()).toContain('等待人工确认后继续执行')
  })

  it('retries failed release from detail action', async () => {
    getReleaseDetail.mockResolvedValue({
      id: 'REL-20260607-031',
      sourceBaselineId: 'BL-20260607-0001',
      targetEnvironmentName: '项目 X 生产',
      status: 'FAILED',
      progress: 100,
      agentName: 'agent-project-x',
      agentTaskId: 'AGT-REL-031',
      steps: [
        { id: 'release-step-1', name: '构建产物校验', status: 'SUCCESS', message: '' },
        { id: 'release-step-2', name: '灰度发布', status: 'FAILED', message: '' },
      ],
      failures: [{ serviceName: 'svc-a', reason: 'hook failed', suggestion: 'retry' }],
      actionRecords: [{ action: 'FAIL_FAST', operator: 'system', status: 'FAILED', message: 'failed', occurredAt: '2026-06-08T10:00:00Z' }],
      report: {
        generatedAt: '2026-06-08T10:10:00Z',
        operator: 'li.si',
        successServiceCount: 1,
        failedServiceCount: 1,
        manualConfirmCount: 0,
        rollbackRecommended: true,
        summary: 'need rollback',
      },
      logs: ['[ERROR] release failed'],
    })
    getAgentTaskStatus.mockResolvedValue({
      enabled: true,
      status: {
        taskId: 'AGT-REL-031',
        type: 'release',
        step: '灰度发布',
        status: 'FAILED',
        updatedAt: '2026-06-08T10:00:00Z',
      },
      logs: ['[ERROR] failed'],
    })
    retryRelease.mockResolvedValue({
      releaseId: 'REL-20260607-031',
      action: 'retry',
      status: 'RUNNING',
      message: '已提交失败重试',
    })

    const wrapper = mount(ReleaseDetailPage, {
      global: {
        stubs: {
          DeployStepPanel: { template: '<div data-testid="step-panel" />', props: ['title', 'status', 'steps', 'activeStepName'] },
          LogTerminal: { template: '<div data-testid="log-terminal">{{ title }}|{{ badge }}</div>', props: ['title', 'logs', 'badge'] },
          MetricCard: { template: '<div class="metric-card">{{ label }}:{{ value }}|{{ foot }}|{{ tone }}</div>', props: ['label', 'value', 'foot', 'tone'] },
          ServiceFailureDrawer: { template: '<div data-testid="drawer" />', props: ['visible', 'failure'] },
          StatusTag: { template: '<span class="status-tag">{{ status }}</span>', props: ['status'] },
          ElButton: { template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>', props: ['disabled', 'type', 'link', 'loading'] },
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
          ElDialog: { template: '<div><slot /></div>', props: ['modelValue', 'title', 'width'] },
          ElTable: { template: '<div><slot /></div>', props: ['data', 'class'] },
          ElTableColumn: { template: '<div><slot :row="{}" /></div>', props: ['prop', 'label', 'minWidth', 'fixed', 'width'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()
    await wrapper.get('button:not([disabled])').trigger('click')
    await flushPromises()

    expect(retryRelease).toHaveBeenCalledWith('REL-20260607-031')
    expect(messageSuccess).toHaveBeenCalledWith('已提交失败重试')
  })

  it('shows release report trigger and action records from detail', async () => {
    getReleaseDetail.mockResolvedValue({
      id: 'REL-20260607-031',
      sourceBaselineId: 'BL-20260607-0001',
      targetEnvironmentName: '项目 X 生产',
      status: 'PARTIAL_FAILED',
      progress: 81,
      agentName: 'agent-project-x',
      agentTaskId: 'AGT-REL-031',
      steps: [{ name: '灰度发布', status: 'FAILED', message: '' }],
      failures: [{ serviceName: 'svc-a', reason: 'hook failed', suggestion: 'retry' }],
      actionRecords: [{ action: 'AUTO_RETRY', operator: 'system', status: 'SUCCESS', message: 'retried', occurredAt: '2026-06-08T10:00:00Z' }],
      report: {
        generatedAt: '2026-06-08T10:10:00Z',
        operator: 'li.si',
        successServiceCount: 66,
        failedServiceCount: 2,
        manualConfirmCount: 1,
        rollbackRecommended: true,
        summary: 'summary text',
      },
      logs: ['[ERROR] release failed'],
    })
    getAgentTaskStatus.mockResolvedValue({
      enabled: true,
      status: {
        taskId: 'AGT-REL-031',
        type: 'release',
        step: '灰度发布',
        status: 'PARTIAL_FAILED',
        updatedAt: '2026-06-08T10:00:00Z',
      },
      logs: ['[ERROR] failed'],
    })

    const wrapper = mount(ReleaseDetailPage, {
      global: {
        stubs: {
          DeployStepPanel: { template: '<div data-testid="step-panel" />', props: ['title', 'status', 'steps', 'activeStepName'] },
          LogTerminal: { template: '<div data-testid="log-terminal">{{ title }}|{{ badge }}</div>', props: ['title', 'logs', 'badge'] },
          MetricCard: { template: '<div class="metric-card">{{ label }}:{{ value }}|{{ foot }}|{{ tone }}</div>', props: ['label', 'value', 'foot', 'tone'] },
          ServiceFailureDrawer: { template: '<div data-testid="drawer" />', props: ['visible', 'failure'] },
          StatusTag: { template: '<span class="status-tag">{{ status }}</span>', props: ['status'] },
          ElButton: { template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>', props: ['disabled', 'type', 'link', 'loading'] },
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
          ElDialog: { template: '<div><slot /></div>', props: ['modelValue', 'title', 'width'] },
          ElTable: { template: '<div><slot /></div>', props: ['data', 'class'] },
          ElTableColumn: { template: '<div><slot :row="{ status: \'SUCCESS\' }" /></div>', props: ['prop', 'label', 'minWidth', 'fixed', 'width'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('查看发布报告')
    expect(wrapper.text()).toContain('执行记录')
  })
})
