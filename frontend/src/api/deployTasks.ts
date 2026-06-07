import { mockData } from './mockData'

export function listDeployTasks() {
  return Promise.resolve(mockData.deployTasks)
}

export function getDeployTaskDetail() {
  return Promise.resolve(mockData.deployDetail)
}
