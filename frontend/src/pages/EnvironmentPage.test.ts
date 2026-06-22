import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { listEnvironments, createEnvironment, updateEnvironment, checkEnvironment, probeEnvironment } = vi.hoisted(() => ({
  listEnvironments: vi.fn(),
  createEnvironment: vi.fn(),
  updateEnvironment: vi.fn(),
  checkEnvironment: vi.fn(),
  probeEnvironment: vi.fn(),
}))

const { listAgents } = vi.hoisted(() => ({
  listAgents: vi.fn(),
}))

const { listProjects } = vi.hoisted(() => ({
  listProjects: vi.fn(),
}))

const {
  listKubernetesClusters,
  listHarborRegistries,
  listJenkinsInstances,
  refreshKubernetesCluster,
  refreshHarborRegistry,
  refreshJenkinsInstance,
} = vi.hoisted(() => ({
  listKubernetesClusters: vi.fn(),
  listHarborRegistries: vi.fn(),
  listJenkinsInstances: vi.fn(),
  refreshKubernetesCluster: vi.fn(),
  refreshHarborRegistry: vi.fn(),
  refreshJenkinsInstance: vi.fn(),
}))

vi.mock('@/api/environments', () => ({
  listEnvironments,
  createEnvironment,
  updateEnvironment,
  checkEnvironment,
  probeEnvironment,
}))

vi.mock('@/api/agents', () => ({
  listAgents,
}))

vi.mock('@/api/projects', () => ({
  listProjects,
}))

vi.mock('@/api/integrationResources', () => ({
  listKubernetesClusters,
  listHarborRegistries,
  listJenkinsInstances,
  refreshKubernetesCluster,
  refreshHarborRegistry,
  refreshJenkinsInstance,
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
    probeEnvironment.mockReset()
    listAgents.mockReset()
    listProjects.mockReset()
    listKubernetesClusters.mockReset()
    listHarborRegistries.mockReset()
    listJenkinsInstances.mockReset()
    refreshKubernetesCluster.mockReset()
    refreshHarborRegistry.mockReset()
    refreshJenkinsInstance.mockReset()
    listKubernetesClusters.mockResolvedValue([])
    listHarborRegistries.mockResolvedValue([])
    listJenkinsInstances.mockResolvedValue([])
    refreshKubernetesCluster.mockResolvedValue({})
    refreshHarborRegistry.mockResolvedValue({})
    refreshJenkinsInstance.mockResolvedValue({})
    probeEnvironment.mockResolvedValue({})
    listAgents.mockResolvedValue([
      {
        id: 'agent-project-x',
        name: 'agent-project-x',
        environmentId: 'env-project-x-prod',
        environmentName: '项目 X 生产',
        version: 'dev',
        status: 'ONLINE',
        claimStatus: 'CLAIMED',
        capabilities: ['remote-probe'],
        runtimeStatus: {
          kubernetes: { status: 'HEALTHY', message: '远程 K8s 正常', updatedAt: '', items: ['project-x-ns'] },
          harbor: { status: 'HEALTHY', message: '远程 Harbor 正常', updatedAt: '', items: ['project-x-runtime'] },
        },
        lastHeartbeatAt: '',
        currentTaskId: null,
      },
    ])
    listProjects.mockResolvedValue([])
    listEnvironments.mockResolvedValue([
      {
        id: 'env-local-prod',
        name: '本地生产环境',
        code: 'local-prod',
        projectId: '',
        projectName: '',
        productStatus: 'UNBOUND',
        type: 'LOCAL',
        deployTargetType: 'KUBERNETES',
        networkMode: 'DIRECT',
        status: 'HEALTHY',
        agentStatus: 'NOT_REQUIRED',
        lastCheckAt: '2026-06-07T12:40:00+08:00',
        bindings: [],
      },
      {
        id: 'env-project-x-prod',
        name: '项目 X 生产',
        code: 'project-x-prod',
        projectId: 'project-x',
        projectName: '项目 X',
        productStatus: 'BOUND',
        type: 'PROJECT',
        deployTargetType: 'KUBERNETES',
        networkMode: 'AGENT',
        registryId: 'harbor-local',
        registryProject: 'project-x',
        jenkinsId: 'jenkins-local',
        jenkinsView: 'project-x',
        status: 'HEALTHY',
        agentStatus: 'ONLINE',
        lastCheckAt: '2026-06-07T12:40:12+08:00',
        bindings: [
          {
            resourceType: 'HARBOR',
            bindingRole: 'BUILD_SOURCE',
            resourceId: 'harbor-local',
            scopeType: 'PROJECT',
            scopeValue: 'project-x',
            isDefault: true,
          },
          {
            resourceType: 'JENKINS',
            bindingRole: 'BUILD_SOURCE',
            resourceId: 'jenkins-local',
            scopeType: 'VIEW',
            scopeValue: 'project-x',
            isDefault: true,
          },
          {
            resourceType: 'K8S',
            bindingRole: 'RUNTIME_TARGET',
            resourceId: 'agent-runtime-k8s',
            scopeType: 'NAMESPACE',
            scopeValue: 'project-x-ns',
            isDefault: true,
          },
          {
            resourceType: 'HARBOR',
            bindingRole: 'RUNTIME_TARGET',
            resourceId: 'agent-runtime-harbor',
            scopeType: 'PROJECT',
            scopeValue: 'project-x-runtime',
            isDefault: true,
          },
        ],
      },
      {
        id: 'env-project-z-prod',
        name: '项目 Z 生产',
        code: 'project-z-prod',
        projectId: '',
        projectName: '',
        productStatus: 'UNBOUND',
        type: 'PROJECT',
        deployTargetType: 'KUBERNETES',
        networkMode: 'AGENT',
        registryId: 'harbor-local',
        registryProject: 'project-z',
        jenkinsId: 'jenkins-local',
        jenkinsView: 'project-z',
        status: 'UNKNOWN',
        agentStatus: 'OFFLINE',
        lastCheckAt: '2026-06-07T12:31:00+08:00',
        bindings: [],
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
    expect(wrapper.text()).toContain('本地产品绑定本地 K8s、Harbor、Jenkins')
    expect(wrapper.text()).toContain('远程产品绑定本地 Harbor、Jenkins')
    expect(wrapper.text()).toContain('无需 Agent')
    expect(wrapper.text()).toContain('ONLINE')
    expect(wrapper.text()).toContain('本地 Harbor')
    expect(wrapper.text()).toContain('远程 Harbor')
    expect(wrapper.text()).toContain('1 个远程产品 Agent 未就绪')
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

  it('creates remote environments with local build sources and remote runtime scopes', async () => {
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
    vm.form.jenkinsId = 'jenkins-local'
    vm.form.jenkinsViews = ['project-x', 'project-y']
    vm.form.runtimeNamespaces = ['project-x-ns']
    vm.form.runtimeRegistryProjects = ['project-x-runtime']
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
            bindingRole: 'BUILD_SOURCE',
            resourceId: 'harbor-local',
            scopeType: 'PROJECT',
            scopeValue: 'project-x',
            isDefault: true,
          },
          {
            resourceType: 'HARBOR',
            bindingRole: 'BUILD_SOURCE',
            resourceId: 'harbor-local',
            scopeType: 'PROJECT',
            scopeValue: 'project-y',
            isDefault: false,
          },
          {
            resourceType: 'JENKINS',
            bindingRole: 'BUILD_SOURCE',
            resourceId: 'jenkins-local',
            scopeType: 'VIEW',
            scopeValue: 'project-x',
            isDefault: true,
          },
          {
            resourceType: 'JENKINS',
            bindingRole: 'BUILD_SOURCE',
            resourceId: 'jenkins-local',
            scopeType: 'VIEW',
            scopeValue: 'project-y',
            isDefault: false,
          },
          {
            resourceType: 'K8S',
            bindingRole: 'RUNTIME_TARGET',
            resourceId: 'agent-runtime-k8s',
            scopeType: 'NAMESPACE',
            scopeValue: 'project-x-ns',
            isDefault: true,
          },
          {
            resourceType: 'HARBOR',
            bindingRole: 'RUNTIME_TARGET',
            resourceId: 'agent-runtime-harbor',
            scopeType: 'PROJECT',
            scopeValue: 'project-x-runtime',
            isDefault: true,
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
    vm.form.jenkinsId = 'jenkins-local'
    vm.form.jenkinsViews = ['view-not-probed']
    vm.form.runtimeNamespaces = ['runtime-ns-not-probed']
    vm.form.runtimeRegistryProjects = ['runtime-project-not-probed']
    await vm.submitEnvironment()

    expect(createEnvironment).toHaveBeenCalledWith(expect.objectContaining({
      registryProject: 'project-not-probed',
      jenkinsId: 'jenkins-local',
      jenkinsView: 'view-not-probed',
    }))
    expect(ElMessage.warning).toHaveBeenCalledWith(expect.stringContaining('未在最近探测结果中发现'))
    expect(ElMessage.warning).toHaveBeenCalledWith(expect.stringContaining('产品已保存，但存在未验证的资源范围'))
  })

  it('shows all local environment missing scopes and allows refreshing related probes', async () => {
    listKubernetesClusters.mockResolvedValue([
      { id: 'k8s-local', name: '本地 k3s', status: 'HEALTHY', namespaces: ['default'] },
    ])
    listHarborRegistries.mockResolvedValue([
      { id: 'harbor-local', name: '本地 Harbor', status: 'HEALTHY', projects: ['project-x'] },
    ])
    listJenkinsInstances.mockResolvedValue([
      { id: 'jenkins-local', name: '本地 Jenkins', status: 'HEALTHY', views: ['project-x'] },
    ])
    listEnvironments.mockResolvedValue([
      {
        id: 'env-local-missing',
        name: '本地缺失环境',
        code: 'local-missing',
        type: 'LOCAL',
        networkMode: 'DIRECT',
        clusterId: 'k8s-local',
        namespace: 'missing-ns',
        registryId: 'harbor-local',
        registryProject: 'project-x',
        jenkinsId: 'jenkins-local',
        jenkinsView: 'missing-view',
        status: 'UNKNOWN',
        agentStatus: 'NOT_REQUIRED',
        lastCheckAt: '',
        bindings: [
          {
            resourceType: 'K8S',
            resourceId: 'k8s-local',
            scopeType: 'NAMESPACE',
            scopeValue: 'missing-ns',
            isDefault: true,
          },
          {
            resourceType: 'HARBOR',
            resourceId: 'harbor-local',
            scopeType: 'PROJECT',
            scopeValue: 'project-x',
            isDefault: true,
          },
          {
            resourceType: 'JENKINS',
            resourceId: 'jenkins-local',
            scopeType: 'VIEW',
            scopeValue: 'missing-view',
            isDefault: true,
          },
        ],
      },
    ])
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

    expect(wrapper.text()).toContain('K8s 命名空间 missing-ns 未在最近探测结果中发现')
    expect(wrapper.text()).toContain('Jenkins 流水线视图 missing-view 未在最近探测结果中发现')
    expect(wrapper.text()).not.toContain('请刷新相关基础资源探测')
    expect(wrapper.text()).toContain('刷新相关探测')

    await wrapper.findAll('button').find((item) => item.text().includes('刷新相关探测'))?.trigger('click')
    await flushPromises()

    expect(refreshKubernetesCluster).toHaveBeenCalledWith('k8s-local')
    expect(refreshJenkinsInstance).toHaveBeenCalledWith('jenkins-local')
    expect(refreshHarborRegistry).not.toHaveBeenCalled()
    expect(ElMessage.success).toHaveBeenCalledWith('相关基础资源探测已刷新')
  })

  it('shows remote environment missing harbor project reason in list and next step in detail', async () => {
    listHarborRegistries.mockResolvedValue([
      { id: 'harbor-local', name: '本地 Harbor', status: 'HEALTHY', projects: ['project-x'] },
    ])
    listEnvironments.mockResolvedValue([
      {
        id: 'env-remote-missing',
        name: '远程缺失环境',
        code: 'remote-missing',
        type: 'PROJECT',
        networkMode: 'AGENT',
        registryId: 'harbor-local',
        registryProject: 'missing-project',
        status: 'UNKNOWN',
        agentStatus: 'ONLINE',
        lastCheckAt: '',
        bindings: [
          {
            resourceType: 'HARBOR',
            resourceId: 'harbor-local',
            scopeType: 'PROJECT',
            scopeValue: 'missing-project',
            isDefault: true,
          },
        ],
      },
    ])
    const wrapper = mount(EnvironmentPage, {
      global: {
        stubs: {
          EnvironmentConfigDrawer: {
            template: '<aside v-if="visible">{{ diagnostics.map((item) => item.nextStep).join(" ") }}</aside>',
            props: ['visible', 'environment', 'resourceName', 'checking', 'diagnostics', 'checkHelpText'],
          },
          StatusTag: { template: '<span>{{ status }}</span>', props: ['status'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Harbor 镜像项目 missing-project 未在最近探测结果中发现')
    expect(wrapper.text()).not.toContain('请到基础资源刷新 Harbor 探测')

    ;(wrapper.vm as unknown as { openDrawer: (row: unknown) => void; environments: unknown[] }).openDrawer(
      (wrapper.vm as unknown as { environments: unknown[] }).environments[0],
    )
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('请到基础资源刷新 Harbor 探测')
  })

  it('keeps remote agent readiness separate from resource readiness', async () => {
    listHarborRegistries.mockResolvedValue([
      { id: 'harbor-local', name: '本地 Harbor', status: 'HEALTHY', projects: ['project-x'] },
    ])
    listJenkinsInstances.mockResolvedValue([
      { id: 'jenkins-local', name: '本地 Jenkins', status: 'HEALTHY', views: ['project-x'] },
    ])
    listEnvironments.mockResolvedValue([
      {
        id: 'env-remote-unbound',
        name: '远程未绑定环境',
        code: 'remote-unbound',
        type: 'PROJECT',
        networkMode: 'AGENT',
        registryId: 'harbor-local',
        registryProject: 'project-x',
        jenkinsId: 'jenkins-local',
        jenkinsView: 'project-x',
        status: 'HEALTHY',
        agentStatus: 'UNBOUND',
        lastCheckAt: '2026-06-21T21:52:59+08:00',
        bindings: [
          {
            resourceType: 'HARBOR',
            resourceId: 'harbor-local',
            scopeType: 'PROJECT',
            scopeValue: 'project-x',
            isDefault: true,
          },
          {
            resourceType: 'JENKINS',
            resourceId: 'jenkins-local',
            scopeType: 'VIEW',
            scopeValue: 'project-x',
            isDefault: true,
          },
        ],
      },
    ])
    const wrapper = mount(EnvironmentPage, {
      global: {
        stubs: {
          EnvironmentConfigDrawer: {
            template: '<aside v-if="visible">{{ diagnostics.map((item) => item.message).join(" ") }}</aside>',
            props: ['visible', 'environment', 'resourceName', 'checking', 'diagnostics', 'checkHelpText'],
          },
          StatusTag: { template: '<span>{{ status }}</span>', props: ['status'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('远程 Agent 未绑定，会影响远程发布/部署执行')
    expect(wrapper.text()).toContain('远程 K8s 未绑定 Agent，无法获取远程资源清单')
    expect(wrapper.text()).toContain('远程 Harbor 未绑定 Agent，无法获取远程资源清单')
    expect(wrapper.text()).toContain('影响远程执行')

    ;(wrapper.vm as unknown as { openDrawer: (row: unknown) => void; environments: unknown[] }).openDrawer(
      (wrapper.vm as unknown as { environments: unknown[] }).environments[0],
    )
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('远程 Agent 未绑定，会影响远程发布/部署执行')
  })

  it('passes local connection test explanation to the detail drawer', async () => {
    listKubernetesClusters.mockResolvedValue([
      { id: 'k8s-local', name: '本地 k3s', status: 'HEALTHY', namespaces: ['default'] },
    ])
    listHarborRegistries.mockResolvedValue([
      { id: 'harbor-local', name: '本地 Harbor', status: 'HEALTHY', projects: ['project-x'] },
    ])
    listEnvironments.mockResolvedValue([
      {
        id: 'env-local-detail',
        name: '本地详情环境',
        code: 'local-detail',
        type: 'LOCAL',
        networkMode: 'DIRECT',
        clusterId: 'k8s-local',
        namespace: 'default',
        registryId: 'harbor-local',
        registryProject: 'project-x',
        status: 'UNKNOWN',
        agentStatus: 'NOT_REQUIRED',
        lastCheckAt: '',
        bindings: [],
      },
    ])
    const wrapper = mount(EnvironmentPage, {
      global: {
        stubs: {
          EnvironmentConfigDrawer: {
            template: '<aside>{{ checkHelpText }} {{ diagnostics.map((item) => item.message).join(" ") }}</aside>',
            props: ['environment', 'resourceName', 'checking', 'diagnostics', 'checkHelpText'],
          },
          StatusTag: { template: '<span>{{ status }}</span>', props: ['status'] },
        },
        directives: {
          loading: () => undefined,
        },
      },
    })

    await flushPromises()
    ;(wrapper.vm as unknown as { openDrawer: (row: unknown) => void; environments: unknown[] }).openDrawer(
      (wrapper.vm as unknown as { environments: unknown[] }).environments[0],
    )
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('本地连接测试：平台后端直接校验已绑定的 K8s、Harbor、Jenkins 范围，不依赖 Agent。')
    expect(wrapper.text()).toContain('K8s 命名空间 default 已在最近探测结果中发现')
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
            bindingRole: 'BUILD_SOURCE',
            resourceId: 'k8s-local',
            scopeType: 'NAMESPACE',
            scopeValue: 'project-x',
            isDefault: true,
          },
          {
            resourceType: 'K8S',
            bindingRole: 'BUILD_SOURCE',
            resourceId: 'k8s-local',
            scopeType: 'NAMESPACE',
            scopeValue: 'default',
            isDefault: false,
          },
          {
            resourceType: 'HARBOR',
            bindingRole: 'BUILD_SOURCE',
            resourceId: 'harbor-local',
            scopeType: 'PROJECT',
            scopeValue: 'project-x',
            isDefault: true,
          },
          {
            resourceType: 'HARBOR',
            bindingRole: 'BUILD_SOURCE',
            resourceId: 'harbor-local',
            scopeType: 'PROJECT',
            scopeValue: 'project-y',
            isDefault: false,
          },
          {
            resourceType: 'JENKINS',
            bindingRole: 'BUILD_SOURCE',
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
    listJenkinsInstances.mockResolvedValue([
      { id: 'jenkins-local', name: '本地 Jenkins', status: 'HEALTHY', views: ['project-x'] },
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
    vm.form.registryProjects = ['project-x']
    vm.form.jenkinsViews = ['project-x']
    vm.form.runtimeNamespaces = ['project-x-ns']
    vm.form.runtimeRegistryProjects = ['project-x-runtime']
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
