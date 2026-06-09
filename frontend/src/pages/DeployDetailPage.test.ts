import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  getDeployTaskDetail,
  getAgentTaskStatus,
  retryDeployStep,
  skipDeployStep,
  confirmDeployStep,
  messageSuccess,
  messageWarning,
} = vi.hoisted(() => ({
  getDeployTaskDetail: vi.fn(),
  getAgentTaskStatus: vi.fn(),
  retryDeployStep: vi.fn(),
  skipDeployStep: vi.fn(),
  confirmDeployStep: vi.fn(),
  messageSuccess: vi.fn(),
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

vi.mock('@/api/deployTasks', () => ({
  getDeployTaskDetail,
  retryDeployStep,
  skipDeployStep,
  confirmDeployStep,
}))

vi.mock('@/api/agentTasks', () => ({
  getAgentTaskStatus,
}))

import DeployDetailPage from './DeployDetailPage.vue'

describe('DeployDetailPage', () => {
  beforeEach(() => {
    getDeployTaskDetail.mockReset()
    getAgentTaskStatus.mockReset()
    retryDeployStep.mockReset()
    skipDeployStep.mockReset()
    confirmDeployStep.mockReset()
    messageSuccess.mockReset()
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
          ElButton: { template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>', props: ['disabled', 'type', 'link', 'loading'] },
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(messageWarning).toHaveBeenCalledWith('加载部署任务详情失败，已显示本地示例数据')
    expect(getAgentTaskStatus).toHaveBeenCalledWith('DEP-20260607-009')
    expect(wrapper.text()).toContain('部署任务详情：DEP-20260607-009')
    expect(wrapper.text()).toContain('Agent 编排')
  })

  it('polls agent status from detail agent task id and shows failed task state', async () => {
    getDeployTaskDetail.mockResolvedValue({
      id: 'DEP-20260607-009',
      productName: '产品 A',
      targetEnvironmentName: '项目 X 生产',
      source: 'BL-20260607-0001',
      status: 'RUNNING',
      progress: 46,
      agentTaskId: 'AGT-DEP-009',
      steps: [
        { order: 1, name: '检查环境连接', type: 'STANDARD', status: 'SUCCESS' },
        { order: 2, name: '恢复 MinIO', type: 'SHELL', status: 'RUNNING' },
      ],
      actionRecords: [{ action: 'STEP_RETRY', operator: 'system', status: 'SUCCESS', message: 'retried', occurredAt: '2026-06-08T10:00:00Z' }],
      logs: ['[INFO] deploy detail snapshot'],
    })
    getAgentTaskStatus.mockResolvedValue({
      enabled: true,
      status: {
        taskId: 'AGT-DEP-009',
        type: 'deploy',
        step: '恢复 MinIO',
        status: 'FAILED',
        updatedAt: '2026-06-08T10:00:00Z',
      },
      logs: ['[ERROR] restore failed'],
    })

    const wrapper = mount(DeployDetailPage, {
      global: {
        stubs: {
          DeployStepPanel: { template: '<div data-testid="step-panel" />', props: ['title', 'status', 'steps', 'activeStepName'] },
          LogTerminal: { template: '<div data-testid="log-terminal">{{ title }}|{{ badge }}</div>', props: ['title', 'logs', 'badge'] },
          MetricCard: { template: '<div class="metric-card">{{ label }}:{{ value }}|{{ foot }}|{{ tone }}</div>', props: ['label', 'value', 'foot', 'tone'] },
          StatusTag: { template: '<span class="status-tag">{{ status }}</span>', props: ['status'] },
          ElButton: { template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>', props: ['disabled', 'type', 'link', 'loading'] },
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
          ElTable: { template: '<div><slot /></div>', props: ['data', 'class'] },
          ElTableColumn: { template: '<div><slot :row="{ status: \'SUCCESS\' }" /></div>', props: ['prop', 'label', 'minWidth', 'fixed', 'width'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(getAgentTaskStatus).toHaveBeenCalledWith('AGT-DEP-009')
    expect(wrapper.text()).toContain('实时 Agent')
    expect(wrapper.text()).toContain('需处理')
    expect(wrapper.text()).toContain('当前任务失败，需处理后重试')
  })

  it('confirms waiting step from detail action', async () => {
    getDeployTaskDetail.mockResolvedValue({
      id: 'DEP-20260607-009',
      productName: '产品 A',
      targetEnvironmentName: '项目 X 生产',
      source: 'BL-20260607-0001',
      status: 'RUNNING',
      progress: 46,
      agentTaskId: 'AGT-DEP-009',
      steps: [
        { id: 'step-1', order: 1, name: '检查环境连接', type: 'STANDARD', status: 'SUCCESS' },
        { id: 'step-2', order: 2, name: '人工验收', type: 'MANUAL_CONFIRM', status: 'RUNNING' },
      ],
      actionRecords: [{ action: 'WAIT_CONFIRM', operator: 'system', status: 'PENDING_CONFIRM', message: 'waiting', occurredAt: '2026-06-08T10:00:00Z' }],
      logs: ['[WARN] waiting confirm'],
    })
    getAgentTaskStatus.mockResolvedValue({
      enabled: true,
      status: {
        taskId: 'AGT-DEP-009',
        type: 'deploy',
        step: '人工验收',
        status: 'WAITING_CONFIRM',
        updatedAt: '2026-06-08T10:00:00Z',
      },
      logs: ['[WARN] waiting confirm'],
    })
    confirmDeployStep.mockResolvedValue({
      taskId: 'DEP-20260607-009',
      stepId: 'step-2',
      action: 'confirm',
      status: 'RUNNING',
      message: '已提交人工确认',
    })

    const wrapper = mount(DeployDetailPage, {
      global: {
        stubs: {
          DeployStepPanel: { template: '<div data-testid="step-panel" />', props: ['title', 'status', 'steps', 'activeStepName'] },
          LogTerminal: { template: '<div data-testid="log-terminal">{{ title }}|{{ badge }}</div>', props: ['title', 'logs', 'badge'] },
          MetricCard: { template: '<div class="metric-card">{{ label }}:{{ value }}|{{ foot }}|{{ tone }}</div>', props: ['label', 'value', 'foot', 'tone'] },
          StatusTag: { template: '<span class="status-tag">{{ status }}</span>', props: ['status'] },
          ElButton: { template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>', props: ['disabled', 'type', 'link', 'loading'] },
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
          ElTable: { template: '<div><slot /></div>', props: ['data', 'class'] },
          ElTableColumn: { template: '<div><slot :row="{ status: \'SUCCESS\' }" /></div>', props: ['prop', 'label', 'minWidth', 'fixed', 'width'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()
    const buttons = wrapper.findAll('button:not([disabled])')
    await buttons[1].trigger('click')
    await flushPromises()

    expect(confirmDeployStep).toHaveBeenCalledWith('DEP-20260607-009', 'step-2')
    expect(messageSuccess).toHaveBeenCalledWith('已提交人工确认')
  })

  it('shows action records section from detail data', async () => {
    getDeployTaskDetail.mockResolvedValue({
      id: 'DEP-20260607-009',
      productName: '产品 A',
      targetEnvironmentName: '项目 X 生产',
      source: 'BL-20260607-0001',
      status: 'RUNNING',
      progress: 46,
      agentTaskId: 'AGT-DEP-009',
      steps: [
        { id: 'step-1', order: 1, name: '检查环境连接', type: 'STANDARD', status: 'SUCCESS' },
        { id: 'step-2', order: 2, name: '恢复 MinIO', type: 'SHELL', status: 'RUNNING' },
      ],
      actionRecords: [{ action: 'STEP_RETRY', operator: 'system', status: 'SUCCESS', message: 'retried', occurredAt: '2026-06-08T10:00:00Z' }],
      auditSummary: {
        operator: 'wang.wu',
        targetEnvironmentName: '项目 X 生产',
        affectedServices: ['order-web', 'payment-worker'],
        result: 'RUNNING',
        failedStep: '',
        lastAction: 'STEP_RETRY',
        lastActionAt: '2026-06-08T10:00:00Z',
      },
      logs: ['[INFO] deploy detail snapshot'],
    })
    getAgentTaskStatus.mockResolvedValue({
      enabled: true,
      status: {
        taskId: 'AGT-DEP-009',
        type: 'deploy',
        step: '恢复 MinIO',
        status: 'RUNNING',
        updatedAt: '2026-06-08T10:00:00Z',
      },
      logs: ['[INFO] running'],
    })

    const wrapper = mount(DeployDetailPage, {
      global: {
        stubs: {
          DeployStepPanel: { template: '<div data-testid="step-panel" />', props: ['title', 'status', 'steps', 'activeStepName'] },
          LogTerminal: { template: '<div data-testid="log-terminal">{{ title }}|{{ badge }}</div>', props: ['title', 'logs', 'badge'] },
          MetricCard: { template: '<div class="metric-card">{{ label }}:{{ value }}|{{ foot }}|{{ tone }}</div>', props: ['label', 'value', 'foot', 'tone'] },
          StatusTag: { template: '<span class="status-tag">{{ status }}</span>', props: ['status'] },
          ElButton: { template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>', props: ['disabled', 'type', 'link', 'loading'] },
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
          ElTable: { template: '<div><slot /></div>', props: ['data', 'class'] },
          ElTableColumn: { template: '<div><slot :row="{ status: \'SUCCESS\' }" /></div>', props: ['prop', 'label', 'minWidth', 'fixed', 'width'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('执行记录')
    expect(wrapper.text()).toContain('审计与影响范围')
    expect(wrapper.text()).toContain('order-web、payment-worker')
    expect(wrapper.text()).toContain('wang.wu')
  })
})
