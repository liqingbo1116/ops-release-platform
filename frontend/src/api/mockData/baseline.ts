import baselineDetail from '../../../../mocks/baseline-detail.json'
import baselines from '../../../../mocks/baselines.json'
import diffResult from '../../../../mocks/diff-result.json'

type BaselineListItem = {
  id: string
  name: string
  sourceEnvironmentId?: string
  sourceEnvironmentName: string
  serviceCount: number
  createdBy: string
  createdAt: string
  status: string
  purpose: string
  lockedAt?: string
}

type BaselineDetailItem = {
  id: string
  name: string
  sourceEnvironmentId?: string
  sourceEnvironmentName: string
  serviceCount: number
  status: string
  createdBy?: string
  createdAt?: string
  purpose?: string
  lockedAt?: string
  items: Array<{
    serviceId: string
    serviceName: string
    namespace: string
    workloadName: string
    workloadType: string
    tag: string
    digest: string
    replicas: number
    readyReplicas: number
    healthStatus: string
  }>
}

export const baselineMockData = {
  baselineDetail: baselineDetail as BaselineDetailItem,
  baselines: baselines as BaselineListItem[],
  diffResult,
}
