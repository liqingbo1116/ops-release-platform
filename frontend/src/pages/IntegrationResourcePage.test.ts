import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  listKubernetesClusters,
  listHarborRegistries,
  listJenkinsInstances,
  createHarborRegistry,
  createJenkinsInstance,
  createKubernetesCluster,
  updateHarborRegistry,
  updateJenkinsInstance,
  updateKubernetesCluster,
  testHarborRegistry,
  testJenkinsInstance,
  testKubernetesCluster,
  refreshHarborRegistry,
  refreshJenkinsInstance,
  refreshKubernetesCluster,
} = vi.hoisted(() => ({
  listKubernetesClusters: vi.fn(),
  listHarborRegistries: vi.fn(),
  listJenkinsInstances: vi.fn(),
  createHarborRegistry: vi.fn(),
  createJenkinsInstance: vi.fn(),
  createKubernetesCluster: vi.fn(),
  updateHarborRegistry: vi.fn(),
  updateJenkinsInstance: vi.fn(),
  updateKubernetesCluster: vi.fn(),
  testHarborRegistry: vi.fn(),
  testJenkinsInstance: vi.fn(),
  testKubernetesCluster: vi.fn(),
  refreshHarborRegistry: vi.fn(),
  refreshJenkinsInstance: vi.fn(),
  refreshKubernetesCluster: vi.fn(),
}))

vi.mock('@/api/integrationResources', () => ({
  listKubernetesClusters,
  listHarborRegistries,
  listJenkinsInstances,
  createHarborRegistry,
  createJenkinsInstance,
  createKubernetesCluster,
  updateHarborRegistry,
  updateJenkinsInstance,
  updateKubernetesCluster,
  testHarborRegistry,
  testJenkinsInstance,
  testKubernetesCluster,
  refreshHarborRegistry,
  refreshJenkinsInstance,
  refreshKubernetesCluster,
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

import IntegrationResourcePage from './IntegrationResourcePage.vue'

describe('IntegrationResourcePage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listKubernetesClusters.mockResolvedValue([
      {
        id: 'k8s-local',
        name: '本地 k3s',
        apiServer: 'https://k8s.example.invalid:6443',
        context: 'default',
        status: 'HEALTHY',
        lastCheckAt: '2026-06-18T10:20:30+08:00',
        probeMessage: 'connected',
        namespaces: ['default'],
      },
    ])
    listHarborRegistries.mockResolvedValue([])
    listJenkinsInstances.mockResolvedValue([])
  })

  it('keeps base resources in a standalone page with simplified create fields', async () => {
    const wrapper = mount(IntegrationResourcePage, {
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
    await wrapper.findAll('button').find((button) => button.text() === '新增资源')?.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('基础资源')
    expect(wrapper.text()).toContain('2026-06-18 10:20')
    expect(wrapper.text()).not.toContain('API Server')
    expect(wrapper.text()).not.toContain('Context')
    expect(wrapper.text()).not.toContain('探测信息')
    expect(wrapper.text()).not.toContain('connected')
    expect(wrapper.text()).toContain('Kubeconfig')
    const dialogText = wrapper.get('.el-dialog').text()
    expect(dialogText).not.toContain('资源 ID')
    expect(dialogText).not.toContain('API Server')
    expect(dialogText).not.toContain('协议')
  })

  it('defaults harbor address without protocol to https when saving', async () => {
    createHarborRegistry.mockResolvedValue({})
    const wrapper = mount(IntegrationResourcePage, {
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
    await wrapper.findAll('.el-tabs__item').find((tab) => tab.text() === 'Harbor 仓库')?.trigger('click')
    await wrapper.findAll('button').find((button) => button.text() === '新增资源')?.trigger('click')
    await wrapper.vm.$nextTick()

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('本地 Harbor')
    await inputs[1].setValue('reg.example.com:5000')
    await inputs[2].setValue('admin')
    await inputs[3].setValue('secret')
    await wrapper.findAll('button').find((button) => button.text() === '保存')?.trigger('click')
    await flushPromises()

    expect(createHarborRegistry).toHaveBeenCalledWith(
      expect.objectContaining({
        url: 'https://reg.example.com:5000',
        scheme: 'https',
      }),
    )
    expect(refreshHarborRegistry).toHaveBeenCalledWith('harbor-harbor')
  })
})
