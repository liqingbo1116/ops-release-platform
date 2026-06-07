import { mockData } from './mockData'

export function listBaselines() {
  return Promise.resolve(mockData.baselines)
}

export function getBaselineDetail() {
  return Promise.resolve(mockData.baselineDetail)
}

export function getBaselineCompare() {
  return Promise.resolve(mockData.diffResult)
}
