import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getData, postData, putData } = vi.hoisted(() => ({
  getData: vi.fn(),
  postData: vi.fn(),
  putData: vi.fn(),
}))

vi.mock('./client', () => ({
  getData,
  postData,
  putData,
}))

import { listEnvironments } from './environments'

describe('environment API', () => {
  beforeEach(() => {
    getData.mockReset()
    postData.mockReset()
    putData.mockReset()
  })

  it('always loads environment list from backend API', async () => {
    getData.mockResolvedValue({
      items: [
        {
          id: 'env-real-prod',
          name: '真实生产环境',
          code: 'real-prod',
          type: 'PROJECT',
          networkMode: 'AGENT',
          status: 'HEALTHY',
          agentStatus: 'ONLINE',
          lastCheckAt: '2026-06-12T10:00:00+08:00',
        },
      ],
      page: 1,
      pageSize: 20,
      total: 1,
    })

    const environments = await listEnvironments()

    expect(getData).toHaveBeenCalledWith('/api/environments')
    expect(environments).toEqual([
      {
        id: 'env-real-prod',
        name: '真实生产环境',
        code: 'real-prod',
        type: 'PROJECT',
        deployTargetType: 'KUBERNETES',
        networkMode: 'AGENT',
        clusterId: '',
        namespace: '',
        registryId: '',
        registryProject: '',
        jenkinsId: '',
        jenkinsView: '',
        status: 'HEALTHY',
        agentStatus: 'ONLINE',
        lastCheckAt: '2026-06-12T10:00:00+08:00',
        bindings: [],
      },
    ])
  })
})
