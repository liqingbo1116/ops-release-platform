import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { listEnvironments, createEnvironment, updateEnvironment, checkEnvironment } = vi.hoisted(() => ({
  listEnvironments: vi.fn(),
  createEnvironment: vi.fn(),
  updateEnvironment: vi.fn(),
  checkEnvironment: vi.fn(),
}))

const { listKubernetesClusters, listHarborRegistries, listJenkinsInstances } = vi.hoisted(() => ({
  listKubernetesClusters: vi.fn(),
  listHarborRegistries: vi.fn(),
  listJenkinsInstances: vi.fn(),
}))

vi.mock('@/api/environments', () => ({
  listEnvironments,
  createEnvironment,
  updateEnvironment,
  checkEnvironment,
}))

vi.mock('@/api/integrationResources', () => ({
  listKubernetesClusters,
  listHarborRegistries,
  listJenkinsInstances,
}))

vi.mock('element-plus', async (importOriginal) => {
  const actual = await importOriginal<typeof import('element-plus')>()
  return {
    ...actual,
    ElMessage: {
      success: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
    },
  }
})

import EnvironmentPage from './EnvironmentPage.vue'
import { ElMessage } from 'element-plus'

describe('EnvironmentPage', () => {
  beforeEach(() => {
    listEnvironments.mockReset()
    createEnvironment.mockReset()
    updateEnvironment.mockReset()
    checkEnvironment.mockReset()
    listKubernetesClusters.mockReset()
    listHarborRegistries.mockReset()
    listJenkinsInstances.mockReset()
    listKubernetesClusters.mockResolvedValue([])
    listHarborRegistries.mockResolvedValue([])
    listJenkinsInstances.mockResolvedValue([])
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
        registryId: 'harbor-local',
        registryProject: 'project-x',
        jenkinsId: 'jenkins-local',
        jenkinsView: 'project-x',
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
        registryId: 'harbor-local',
        registryProject: 'project-z',
        jenkinsId: 'jenkins-local',
        jenkinsView: 'project-z',
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
    expect(wrapper.text()).toContain('本地环境由平台直连基础资源')
    expect(wrapper.text()).toContain('远程环境')
    expect(wrapper.text()).toContain('无需 Agent')
    expect(wrapper.text()).toContain('ONLINE')
    expect(wrapper.text()).toContain('本地环境关联 K8s 命名空间')
    expect(wrapper.text()).toContain('远程环境关联本地 Harbor 镜像项目')
    expect(wrapper.text()).toContain('1 个远程环境 Agent 未就绪')
    expect(wrapper.text()).not.toContain('网络模式')
  })

  it('filters environments by keyword and type', async () => {
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

    ;(wrapper.vm as unknown as { environmentType: string }).environmentType = 'LOCAL'
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).not.toContain('项目 Z 生产')
  })

  it('creates remote environments with local harbor and jenkins scopes', async () => {
    listHarborRegistries.mockResolvedValue([
      { id: 'harbor-local', name: '本地 Harbor', status: 'HEALTHY', projects: ['project-x'] },
    ])
    listJenkinsInstances.mockResolvedValue([
      { id: 'jenkins-local', name: '本地 Jenkins', status: 'HEALTHY', views: ['project-x'] },
    ])
    createEnvironment.mockResolvedValue({
      id: 'env-remote-new-prod',
      name: 'remote new prod',
      code: 'remote-new-prod',
      type: 'PROJECT',
      networkMode: 'AGENT',
      registryId: 'harbor-local',
      registryProject: 'project-x',
      jenkinsId: 'jenkins-local',
      jenkinsView: 'project-x',
      status: 'UNKNOWN',
      agentStatus: 'OFFLINE',
      lastCheckAt: '',
    })
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

    const vm = wrapper.vm as unknown as {
      openCreateDialog: () => void
      submitEnvironment: () => Promise<void>
      form: Record<string, string | string[]>
    }
    vm.openCreateDialog()
    vm.form.name = 'remote new prod'
    await wrapper.vm.$nextTick()
    vm.form.clusterId = 'cluster-should-clear'
    vm.form.namespace = 'namespace-should-clear'
    vm.form.registryProjects = ['project-x', 'project-y']
    vm.form.jenkinsViews = ['project-x', 'project-y']
    await vm.submitEnvironment()

    expect(createEnvironment).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'remote new prod',
        code: 'remote-new-prod',
        type: 'PROJECT',
        networkMode: 'AGENT',
        clusterId: '',
        namespace: '',
        registryId: 'harbor-local',
        registryProject: 'project-x',
        jenkinsId: 'jenkins-local',
        jenkinsView: 'project-x',
        bindings: [
          {
            resourceType: 'HARBOR',
            resourceId: 'harbor-local',
            scopeType: 'PROJECT',
            scopeValue: 'project-x',
            isDefault: true,
          },
          {
            resourceType: 'HARBOR',
            resourceId: 'harbor-local',
            scopeType: 'PROJECT',
            scopeValue: 'project-y',
            isDefault: false,
          },
          {
            resourceType: 'JENKINS',
            resourceId: 'jenkins-local',
            scopeType: 'VIEW',
            scopeValue: 'project-x',
            isDefault: true,
          },
          {
            resourceType: 'JENKINS',
            resourceId: 'jenkins-local',
            scopeType: 'VIEW',
            scopeValue: 'project-y',
            isDefault: false,
          },
        ],
      }),
    )
  })

  it('keeps manual scopes savable and warns when they are not in probe cache', async () => {
    listHarborRegistries.mockResolvedValue([
      { id: 'harbor-local', name: '本地 Harbor', status: 'HEALTHY', projects: ['project-x'] },
    ])
    listJenkinsInstances.mockResolvedValue([
      { id: 'jenkins-local', name: '本地 Jenkins', status: 'HEALTHY', views: ['project-x'] },
    ])
    createEnvironment.mockResolvedValue({
      id: 'env-remote-manual-scope',
      name: 'remote manual scope',
      code: 'remote-manual-scope',
      type: 'PROJECT',
      networkMode: 'AGENT',
      registryId: 'harbor-local',
      registryProject: 'project-not-probed',
      jenkinsId: 'jenkins-local',
      jenkinsView: 'view-not-probed',
      status: 'DEGRADED',
      agentStatus: 'UNBOUND',
      lastCheckAt: '',
      bindings: [],
    })
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

    const vm = wrapper.vm as unknown as {
      openCreateDialog: () => void
      submitEnvironment: () => Promise<void>
      form: Record<string, string | string[]>
    }
    vm.openCreateDialog()
    vm.form.name = 'remote manual scope'
    vm.form.registryProjects = ['project-not-probed']
    vm.form.jenkinsViews = ['view-not-probed']
    await vm.submitEnvironment()

    expect(createEnvironment).toHaveBeenCalledWith(expect.objectContaining({
      registryProject: 'project-not-probed',
      jenkinsView: 'view-not-probed',
    }))
    expect(ElMessage.warning).toHaveBeenCalledWith(expect.stringContaining('未在最近探测结果中发现'))
    expect(ElMessage.warning).toHaveBeenCalledWith(expect.stringContaining('环境已保存，但存在未验证的资源范围'))
  })

  it('creates local environments with selected platform integration resources', async () => {
    listKubernetesClusters.mockResolvedValue([
      { id: 'k8s-local', name: '本地 k3s', status: 'HEALTHY', namespaces: ['default', 'project-x'] },
    ])
    listHarborRegistries.mockResolvedValue([
      { id: 'harbor-local', name: '本地 Harbor', status: 'HEALTHY', projects: ['project-x'] },
    ])
    listJenkinsInstances.mockResolvedValue([
      { id: 'jenkins-local', name: '本地 Jenkins', status: 'HEALTHY', views: ['project-x'] },
    ])
    createEnvironment.mockResolvedValue({
      id: 'env-local-project-x',
      name: '本地项目 X',
      code: 'local-project-x',
      type: 'LOCAL',
      networkMode: 'DIRECT',
      clusterId: 'k8s-local',
      namespace: 'project-x',
      registryId: 'harbor-local',
      registryProject: 'project-x',
      jenkinsId: 'jenkins-local',
      jenkinsView: 'project-x',
      status: 'HEALTHY',
      agentStatus: 'NOT_REQUIRED',
      lastCheckAt: '',
    })
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

    const vm = wrapper.vm as unknown as {
      openCreateDialog: () => void
      submitEnvironment: () => Promise<void>
      form: Record<string, string | string[]>
    }
    vm.openCreateDialog()
    vm.form.type = 'LOCAL'
    await wrapper.vm.$nextTick()
    vm.form.name = '本地项目 X'
    vm.form.namespaces = ['project-x', 'default']
    vm.form.registryProjects = ['project-x', 'project-y']
    vm.form.jenkinsViews = ['project-x']
    await vm.submitEnvironment()

    expect(createEnvironment).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'LOCAL',
        code: expect.stringMatching(/^local-\d{14}$/),
        networkMode: 'DIRECT',
        clusterId: 'k8s-local',
        namespace: 'project-x',
        registryId: 'harbor-local',
        registryProject: 'project-x',
        jenkinsId: 'jenkins-local',
        jenkinsView: 'project-x',
        bindings: [
          {
            resourceType: 'K8S',
            resourceId: 'k8s-local',
            scopeType: 'NAMESPACE',
            scopeValue: 'project-x',
            isDefault: true,
          },
          {
            resourceType: 'K8S',
            resourceId: 'k8s-local',
            scopeType: 'NAMESPACE',
            scopeValue: 'default',
            isDefault: false,
          },
          {
            resourceType: 'HARBOR',
            resourceId: 'harbor-local',
            scopeType: 'PROJECT',
            scopeValue: 'project-x',
            isDefault: true,
          },
          {
            resourceType: 'HARBOR',
            resourceId: 'harbor-local',
            scopeType: 'PROJECT',
            scopeValue: 'project-y',
            isDefault: false,
          },
          {
            resourceType: 'JENKINS',
            resourceId: 'jenkins-local',
            scopeType: 'VIEW',
            scopeValue: 'project-x',
            isDefault: true,
          },
        ],
      }),
    )
  })

  it('generates environment identifiers from the environment name', async () => {
    listHarborRegistries.mockResolvedValue([
      { id: 'harbor-local', name: '本地 Harbor', status: 'HEALTHY', projects: ['project-x'] },
    ])
    createEnvironment.mockResolvedValue({
      id: 'env-remote-prod',
      name: 'remote prod',
      code: 'remote-prod',
      type: 'PROJECT',
      networkMode: 'AGENT',
      registryId: 'harbor-local',
      registryProject: 'project-x',
      status: 'UNKNOWN',
      agentStatus: 'UNBOUND',
      lastCheckAt: '',
    })
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

    const vm = wrapper.vm as unknown as {
      openCreateDialog: () => void
      submitEnvironment: () => Promise<void>
      form: Record<string, string | string[]>
    }
    vm.openCreateDialog()
    vm.form.name = 'remote prod'
    vm.form.registryProject = 'project-x'
    await wrapper.vm.$nextTick()
    expect(vm.form.code).toBe('remote-prod')

    await vm.submitEnvironment()
    expect(createEnvironment).toHaveBeenCalledWith(expect.objectContaining({ code: 'remote-prod' }))
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
