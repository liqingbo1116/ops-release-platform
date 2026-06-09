import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const push = vi.fn()
const { listDeployTasks, messageWarning } = vi.hoisted(() => ({
  listDeployTasks: vi.fn(),
  messageWarning: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}))

vi.mock('element-plus', async (importOriginal) => {
  const actual = await importOriginal<typeof import('element-plus')>()
  return {
    ...actual,
    ElMessage: {
      ...actual.ElMessage,
      warning: messageWarning,
    },
  }
})

vi.mock('@/api/deployTasks', () => ({
  listDeployTasks,
}))

import DeployListPage from './DeployListPage.vue'

describe('DeployListPage', () => {
  beforeEach(() => {
    push.mockReset()
    listDeployTasks.mockReset()
    messageWarning.mockReset()
    listDeployTasks.mockResolvedValue([
      {
        id: 'DEP-MISSING-RUNNING',
        type: 'SERVICE_DEPLOYMENT',
        productName: '项目 X',
        targetEnvironmentName: '项目 X 生产',
        sourceBaselineId: 'BL-20260607-0001',
        source: 'BL-20260607-0001',
        missingServiceCount: 2,
        serviceNames: ['order-web', 'payment-worker'],
        currentStep: '恢复 MinIO',
        progress: 46,
        status: 'RUNNING',
        agentName: 'agent-project-x',
        agentTaskId: 'AGT-DEP-001',
        nextAction: '等待人工确认数据恢复结果',
      },
      {
        id: 'DEP-MISSING-FAILED',
        type: 'SERVICE_DEPLOYMENT',
        productName: '项目 Z',
        targetEnvironmentName: '项目 Z 生产',
        sourceBaselineId: 'BL-20260605-0011',
        source: 'BL-20260605-0011',
        missingServiceCount: 1,
        serviceNames: ['billing-job'],
        currentStep: '应用 manifests',
        progress: 62,
        status: 'FAILED',
        agentName: 'agent-project-z',
        agentTaskId: 'AGT-DEP-002',
      },
    ])
  })

  it('shows missing-service first deployments from user view', async () => {
    const wrapper = mount(DeployListPage, {
      global: {
        stubs: {
          StatusTag: { template: '<span>{{ status }}</span>', props: ['status'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('服务部署任务')
    expect(wrapper.text()).toContain('BL-20260607-0001')
    expect(wrapper.text()).toContain('缺失服务首次部署')
    expect(wrapper.text()).toContain('2 个目标缺失服务')
    expect(wrapper.text()).toContain('order-web、payment-worker')
    expect(wrapper.text()).toContain('agent-project-x')
    expect(wrapper.text()).toContain('等待人工确认数据恢复结果')
    expect(wrapper.text()).toContain('处理失败后重试当前步骤')
  })

  it('filters by missing service and agent metadata', async () => {
    const wrapper = mount(DeployListPage, {
      global: {
        stubs: {
          StatusTag: { template: '<span>{{ status }}</span>', props: ['status'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    await wrapper.get('input').setValue('billing-job')
    expect(wrapper.text()).toContain('DEP-MISSING-FAILED')
    expect(wrapper.text()).not.toContain('DEP-MISSING-RUNNING')

    await wrapper.get('input').setValue('agent-project-x')
    expect(wrapper.text()).toContain('DEP-MISSING-RUNNING')
    expect(wrapper.text()).not.toContain('DEP-MISSING-FAILED')
  })
})
