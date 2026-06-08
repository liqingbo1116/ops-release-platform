import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const push = vi.fn()
const {
  listAgents,
  listEnvironments,
  getBaselineDetail,
  getBaselineCompare,
  createRelease,
  createDeployTask,
  messageError,
  messageSuccess,
  messageWarning,
} = vi.hoisted(() => ({
  listAgents: vi.fn(),
  listEnvironments: vi.fn(),
  getBaselineDetail: vi.fn(),
  getBaselineCompare: vi.fn(),
  createRelease: vi.fn(),
  createDeployTask: vi.fn(),
  messageError: vi.fn(),
  messageSuccess: vi.fn(),
  messageWarning: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: {
      baselineId: 'BL-20260607-0001',
      targetEnvironmentId: 'env-project-x-prod',
      mode: 'SERVICE_RELEASE',
      serviceIds: '',
    },
    fullPath: '/releases/create?baselineId=BL-20260607-0001&targetEnvironmentId=env-project-x-prod&mode=SERVICE_RELEASE',
  }),
  useRouter: () => ({ push }),
}))

vi.mock('element-plus', () => ({
  ElButton: { template: '<button />' },
  ElCard: { template: '<div><slot name="header" /><slot /></div>' },
  ElForm: { template: '<form><slot /></form>' },
  ElFormItem: { template: '<div><slot /></div>' },
  ElInput: { template: '<input />' },
  ElOption: { template: '<div />' },
  ElRadioButton: { template: '<div><slot /></div>' },
  ElRadioGroup: { template: '<div><slot /></div>' },
  ElSelect: { template: '<div><slot /></div>' },
  ElTag: { template: '<span><slot /></span>' },
  ElMessage: {
    error: messageError,
    success: messageSuccess,
    warning: messageWarning,
  },
}))

vi.mock('@/api/agents', () => ({
  listAgents,
}))

vi.mock('@/api/environments', () => ({
  listEnvironments,
}))

vi.mock('@/api/baselines', () => ({
  getBaselineDetail,
  getBaselineCompare,
}))

vi.mock('@/api/releases', async () => {
  const actual = await vi.importActual<typeof import('@/api/releases')>('@/api/releases')
  return {
    ...actual,
    createRelease,
  }
})

vi.mock('@/api/deployTasks', async () => {
  const actual = await vi.importActual<typeof import('@/api/deployTasks')>('@/api/deployTasks')
  return {
    ...actual,
    createDeployTask,
  }
})

import CreateReleasePage from './CreateReleasePage.vue'

describe('CreateReleasePage submit guard', () => {
  beforeEach(() => {
    push.mockReset()
    listAgents.mockReset()
    listEnvironments.mockReset()
    getBaselineDetail.mockReset()
    getBaselineCompare.mockReset()
    createRelease.mockReset()
    createDeployTask.mockReset()
    messageError.mockReset()
    messageSuccess.mockReset()
    messageWarning.mockReset()

    listAgents.mockResolvedValue([
      {
        id: 'agent-project-x',
        name: 'agent-project-x',
        environmentId: 'env-project-x-prod',
        status: 'ONLINE',
      },
    ])
    listEnvironments.mockResolvedValue([
      {
        id: 'env-project-x-prod',
        name: '项目 X 生产',
        code: 'project-x-prod',
      },
    ])
    getBaselineDetail.mockResolvedValue({
      id: 'BL-20260607-0001',
      name: 'baseline-1',
      sourceEnvironmentName: '项目 X 预发',
    })
    getBaselineCompare.mockResolvedValue({
      sourceBaselineId: 'BL-20260607-0001',
      targetEnvironmentId: 'env-project-x-prod',
      items: [
        {
          serviceId: 'svc-project-x-order',
          serviceName: 'order-service',
          namespace: 'project-x',
          sourceTag: 'v1.0.0',
          targetTag: 'v0.9.0',
          diffStatus: 'TAG_DIFF',
          publishable: false,
          strategy: 'MANUAL_CONFIRM',
        },
      ],
    })
  })

  it('blocks submission when no services are selected', async () => {
    const wrapper = mount(CreateReleasePage, {
      global: {
        stubs: {
          ReleaseRiskPanel: { template: '<div />', props: ['options', 'selectedCount'] },
          ServiceDiffTable: { template: '<div />', props: ['items', 'selectedIds'] },
        },
      },
    })

    await flushPromises()
    await nextTick()

    const submitButton = wrapper.find('button')

    expect(submitButton.attributes('disabled')).toBeDefined()

    await submitButton.trigger('click')

    expect(messageWarning).not.toHaveBeenCalled()
    expect(createRelease).not.toHaveBeenCalled()
    expect(createDeployTask).not.toHaveBeenCalled()
    expect(push).not.toHaveBeenCalled()
  })
})
