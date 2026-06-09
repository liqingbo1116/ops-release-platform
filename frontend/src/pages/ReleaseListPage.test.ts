import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const push = vi.fn()
const { listReleases, messageWarning } = vi.hoisted(() => ({
  listReleases: vi.fn(),
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

vi.mock('@/api/releases', () => ({
  listReleases,
}))

import ReleaseListPage from './ReleaseListPage.vue'

describe('ReleaseListPage', () => {
  beforeEach(() => {
    push.mockReset()
    listReleases.mockReset()
    messageWarning.mockReset()
    listReleases.mockResolvedValue([
      {
        id: 'REL-JENKINS',
        type: 'SERVICE_RELEASE',
        releaseSource: 'JENKINS_JOB',
        buildId: 'BUILD-001',
        targetEnvironmentName: '项目 X 生产',
        agentName: 'agent-project-x',
        progress: 30,
        status: 'JENKINS_QUEUED',
      },
      {
        id: 'REL-HARBOR',
        type: 'SERVICE_RELEASE',
        releaseSource: 'LOCAL_HARBOR_IMAGE',
        imageRepository: 'harbor.local/project-x/user-service',
        imageTag: '20260607-a1b2c3',
        targetEnvironmentName: '项目 X 生产',
        agentName: 'agent-project-x',
        progress: 60,
        status: 'PENDING_IMAGE_SYNC',
      },
      {
        id: 'DEP-MISSING',
        type: 'SERVICE_DEPLOYMENT',
        sourceBaselineId: 'BL-20260607-0001',
        targetEnvironmentName: '项目 X 生产',
        agentName: 'agent-project-x',
        progress: 10,
        status: 'PENDING',
      },
    ])
  })

  it('shows release sources and missing-service deployment source from user view', async () => {
    const wrapper = mount(ReleaseListPage, {
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

    expect(wrapper.text()).toContain('Jenkins Job')
    expect(wrapper.text()).toContain('BUILD-001')
    expect(wrapper.text()).toContain('本地 Harbor 镜像')
    expect(wrapper.text()).toContain('harbor.local/project-x/user-service:20260607-a1b2c3')
    expect(wrapper.text()).toContain('BL-20260607-0001')
    expect(wrapper.text()).toContain('缺失服务首次部署')
  })

  it('filters by Harbor image metadata and deployment wording', async () => {
    const wrapper = mount(ReleaseListPage, {
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

    await wrapper.get('input').setValue('a1b2c3')
    expect(wrapper.text()).toContain('REL-HARBOR')
    expect(wrapper.text()).not.toContain('REL-JENKINS')
    expect(wrapper.text()).not.toContain('DEP-MISSING')

    await wrapper.get('input').setValue('缺失服务')
    expect(wrapper.text()).toContain('DEP-MISSING')
    expect(wrapper.text()).not.toContain('REL-HARBOR')
  })
})
