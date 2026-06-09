import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiClientError } from '@/api/client'
import { useAuthStore } from '@/stores/authStore'

const push = vi.fn()
const {
  routeState,
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
  routeState: {
    query: {
      baselineId: 'BL-20260607-0001',
      targetEnvironmentId: 'env-project-x-prod',
      mode: 'SERVICE_RELEASE',
      serviceIds: '',
    },
    fullPath: '/releases/create?baselineId=BL-20260607-0001&targetEnvironmentId=env-project-x-prod&mode=SERVICE_RELEASE',
  },
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
  useRoute: () => routeState,
  useRouter: () => ({ push }),
}))

vi.mock('element-plus', () => ({
  ElAlert: {
    props: ['type', 'title', 'description'],
    template: '<div data-test="readiness-alert">{{ title }} {{ description }}</div>',
  },
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

function mountPage() {
  const pinia = createPinia()
  setActivePinia(pinia)
  localStorage.setItem('ops-release-token', 'mock-token-admin')

  return mount(CreateReleasePage, {
    global: {
      plugins: [pinia],
      stubs: {
        ElAlert: {
          props: ['type', 'title', 'description'],
          template: '<div data-test="readiness-alert">{{ title }} {{ description }}</div>',
        },
        ReleaseRiskPanel: { template: '<div />', props: ['options', 'selectedCount'] },
        ServiceDiffTable: { template: '<div />', props: ['items', 'selectedIds'] },
      },
    },
  })
}

describe('CreateReleasePage submit flow', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    push.mockReset()
    routeState.query = {
      baselineId: 'BL-20260607-0001',
      targetEnvironmentId: 'env-project-x-prod',
      mode: 'SERVICE_RELEASE',
      serviceIds: '',
    }
    routeState.fullPath = '/releases/create?baselineId=BL-20260607-0001&targetEnvironmentId=env-project-x-prod&mode=SERVICE_RELEASE'
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
      summary: {
        consistent: 1,
        needUpdate: 1,
        missingInTarget: 1,
        workloadError: 0,
        publishable: 2,
      },
      items: [
        {
          serviceId: 'svc-project-x-order',
          serviceName: 'order-service',
          namespace: 'project-x',
          sourceTag: 'v1.0.0',
          targetTag: 'v0.9.0',
          diffStatus: 'NEED_UPDATE',
          publishable: true,
          strategy: 'AUTO',
        },
        {
          serviceId: 'svc-project-x-web',
          serviceName: 'web-service',
          namespace: 'project-x',
          sourceTag: 'v1.0.0',
          targetTag: '',
          diffStatus: 'MISSING_IN_TARGET',
          publishable: true,
          strategy: 'AUTO',
        },
      ],
    })
  })

  it('blocks submission when no services are selected', async () => {
    getBaselineCompare.mockResolvedValue({
      sourceBaselineId: 'BL-20260607-0001',
      targetEnvironmentId: 'env-project-x-prod',
      summary: {
        consistent: 1,
        needUpdate: 0,
        missingInTarget: 1,
        workloadError: 0,
        publishable: 1,
      },
      items: [
        {
          serviceId: 'svc-project-x-web',
          serviceName: 'web-service',
          namespace: 'project-x',
          sourceTag: 'v1.0.0',
          targetTag: '',
          diffStatus: 'MISSING_IN_TARGET',
          publishable: true,
          strategy: 'AUTO',
        },
      ],
    })

    const wrapper = mountPage()

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

  it('shows readiness failure before creating a task when target agent is offline', async () => {
    listAgents.mockResolvedValue([
      {
        id: 'agent-project-z',
        name: 'agent-project-z',
        environmentId: 'env-project-z-prod',
        status: 'OFFLINE',
      },
    ])
    listEnvironments.mockResolvedValue([
      {
        id: 'env-project-z-prod',
        name: '项目 Z 生产',
        code: 'project-z-prod',
      },
    ])
    getBaselineCompare.mockResolvedValue({
      sourceBaselineId: 'BL-20260607-0001',
      targetEnvironmentId: 'env-project-z-prod',
      summary: {
        consistent: 1,
        needUpdate: 1,
        missingInTarget: 0,
        workloadError: 0,
        publishable: 1,
      },
      items: [
        {
          serviceId: 'svc-project-z-order',
          serviceName: 'order-service',
          namespace: 'project-z',
          sourceTag: 'v1.0.0',
          targetTag: 'v0.9.0',
          diffStatus: 'NEED_UPDATE',
          publishable: true,
          strategy: 'AUTO',
        },
      ],
    })
    routeState.query = {
      baselineId: 'BL-20260607-0001',
      targetEnvironmentId: 'env-project-z-prod',
      mode: 'SERVICE_RELEASE',
      serviceIds: '',
    }
    routeState.fullPath = '/releases/create?baselineId=BL-20260607-0001&targetEnvironmentId=env-project-z-prod&mode=SERVICE_RELEASE'

    const wrapper = mountPage()

    await flushPromises()
    await nextTick()

    expect(wrapper.find('[data-test="readiness-alert"]').text()).toContain('目标环境有 1 个 Agent，但当前都不在线')
    expect(wrapper.find('button').attributes('disabled')).toBeDefined()
    expect(createRelease).not.toHaveBeenCalled()
    expect(createDeployTask).not.toHaveBeenCalled()
  })

  it('submits release with NEED_UPDATE services without source baseline id', async () => {
    createRelease.mockResolvedValue({
      id: 'REL-20260608-001',
      status: 'PENDING_CONFIRM',
      agentTaskId: 'AGT-REL-20260608-001',
      createdAt: '2026-06-08T10:00:00Z',
    })

    const wrapper = mountPage()

    await flushPromises()
    await nextTick()

    await wrapper.find('button').trigger('click')

    const releaseRequest = createRelease.mock.calls[0][0]
    expect(releaseRequest).toEqual(expect.objectContaining({
      type: 'SERVICE_RELEASE',
      targetEnvironmentId: 'env-project-x-prod',
      agentId: 'agent-project-x',
      serviceIds: ['svc-project-x-order'],
    }))
    expect(releaseRequest).not.toHaveProperty('sourceBaselineId')
    expect(createDeployTask).not.toHaveBeenCalled()
    expect(messageSuccess).toHaveBeenCalledWith('服务发版已提交 Jenkins')
    expect(push).toHaveBeenCalledWith({
      path: '/releases/REL-20260608-001',
      query: {
        agentTaskId: 'AGT-REL-20260608-001',
      },
    })
  })

  it('submits Harbor image release with image metadata', async () => {
    createRelease.mockResolvedValue({
      id: 'REL-20260608-002',
      status: 'PENDING_IMAGE_SYNC',
      agentTaskId: 'AGT-REL-20260608-002',
      createdAt: '2026-06-08T10:00:00Z',
    })

    const wrapper = mountPage()

    await flushPromises()
    await nextTick()

    ;(wrapper.vm as unknown as { releaseSource: 'JENKINS_JOB' | 'LOCAL_HARBOR_IMAGE' }).releaseSource = 'LOCAL_HARBOR_IMAGE'
    await nextTick()
    await wrapper.find('button').trigger('click')

    expect(createRelease).toHaveBeenCalledWith(expect.objectContaining({
      type: 'SERVICE_RELEASE',
      releaseSource: 'LOCAL_HARBOR_IMAGE',
      targetEnvironmentId: 'env-project-x-prod',
      agentId: 'agent-project-x',
      serviceIds: ['svc-project-x-order'],
      image: {
        repository: 'harbor.local/project-x/order',
        tag: 'v1.0.0',
        digest: 'sha256:mock-v1.0.0',
      },
      jenkins: undefined,
    }))
    expect(createRelease.mock.calls[0][0]).not.toHaveProperty('sourceBaselineId')
    expect(createDeployTask).not.toHaveBeenCalled()
    expect(messageSuccess).toHaveBeenCalledWith('服务发版已提交镜像同步')
    expect(push).toHaveBeenCalledWith({
      path: '/releases/REL-20260608-002',
      query: {
        agentTaskId: 'AGT-REL-20260608-002',
      },
    })
  })

  it('submits deployment with only MISSING_IN_TARGET services', async () => {
    createDeployTask.mockResolvedValue({
      id: 'DEP-20260608-001',
      status: 'PENDING',
      agentTaskId: 'AGT-DEP-20260608-001',
      createdAt: '2026-06-08T10:00:00Z',
    })

    routeState.query = {
      baselineId: 'BL-20260607-0001',
      targetEnvironmentId: 'env-project-x-prod',
      mode: 'SERVICE_DEPLOYMENT',
      serviceIds: 'svc-project-x-web',
    }
    routeState.fullPath = '/releases/create?baselineId=BL-20260607-0001&targetEnvironmentId=env-project-x-prod&mode=SERVICE_DEPLOYMENT&serviceIds=svc-project-x-web'

    const wrapper = mountPage()

    await flushPromises()
    await nextTick()
    await wrapper.find('button').trigger('click')

    expect(createDeployTask).toHaveBeenCalledWith({
      type: 'SERVICE_DEPLOYMENT',
      sourceBaselineId: 'BL-20260607-0001',
      targetEnvironmentId: 'env-project-x-prod',
      agentId: 'agent-project-x',
      serviceIds: ['svc-project-x-web'],
      options: {
        syncImage: true,
        createWorkload: true,
        healthCheck: true,
      },
    })
    expect(createRelease).not.toHaveBeenCalled()
    expect(messageSuccess).toHaveBeenCalledWith('服务部署任务已创建')
    expect(push).toHaveBeenCalledWith({
      path: '/deploy-tasks/DEP-20260608-001',
      query: {
        agentTaskId: 'AGT-DEP-20260608-001',
      },
    })
  })

  it('shows backend diff selection error message', async () => {
    createRelease.mockRejectedValue(new ApiClientError('release services must come from NEED_UPDATE diff items'))

    const wrapper = mountPage()

    await flushPromises()
    await nextTick()
    await wrapper.find('button').trigger('click')

    expect(messageError).toHaveBeenCalledWith('发布单只能提交差异结果中的需更新服务')
  })

  it('blocks release submission locally when user has no release permission', async () => {
    const wrapper = mountPage()

    await flushPromises()
    await nextTick()

    const authStore = useAuthStore()
    authStore.user = {
      id: 'user-viewer',
      username: 'viewer',
      displayName: '只读用户',
      roles: ['VIEWER'],
      permissions: [],
    }
    await nextTick()
    await (wrapper.vm as unknown as { submitRelease: () => Promise<void> }).submitRelease()

    expect(messageWarning).toHaveBeenCalledWith('当前账号没有服务发版权限')
    expect(createRelease).not.toHaveBeenCalled()
    expect(createDeployTask).not.toHaveBeenCalled()
    expect(push).not.toHaveBeenCalled()
  })

  it('blocks deployment submission locally when user has no deploy permission', async () => {
    routeState.query = {
      baselineId: 'BL-20260607-0001',
      targetEnvironmentId: 'env-project-x-prod',
      mode: 'SERVICE_DEPLOYMENT',
      serviceIds: 'svc-project-x-web',
    }
    routeState.fullPath = '/releases/create?baselineId=BL-20260607-0001&targetEnvironmentId=env-project-x-prod&mode=SERVICE_DEPLOYMENT&serviceIds=svc-project-x-web'

    const wrapper = mountPage()

    await flushPromises()
    await nextTick()

    const authStore = useAuthStore()
    authStore.user = {
      id: 'user-viewer',
      username: 'viewer',
      displayName: '只读用户',
      roles: ['VIEWER'],
      permissions: [],
    }
    await nextTick()
    await (wrapper.vm as unknown as { submitRelease: () => Promise<void> }).submitRelease()

    expect(messageWarning).toHaveBeenCalledWith('当前账号没有服务部署权限')
    expect(createRelease).not.toHaveBeenCalled()
    expect(createDeployTask).not.toHaveBeenCalled()
    expect(push).not.toHaveBeenCalled()
  })

  it('shows permission failure returned by backend', async () => {
    createRelease.mockRejectedValue(new ApiClientError('permission denied', { status: 403, code: 'FORBIDDEN' }))

    const wrapper = mountPage()

    await flushPromises()
    await nextTick()
    await wrapper.find('button').trigger('click')

    expect(messageError).toHaveBeenCalledWith('当前账号没有服务发版权限')
    expect(push).not.toHaveBeenCalled()
  })
})
