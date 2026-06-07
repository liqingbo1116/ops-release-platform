import { mockData } from './mockData'

export function listEnvironments() {
  return Promise.resolve(mockData.environments)
}
