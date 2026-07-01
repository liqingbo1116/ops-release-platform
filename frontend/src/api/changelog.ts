import { getData, type PageResult } from './client'

export type ChangelogEntry = {
  id: string
  title: string
  type: string
  description?: string
  author?: string
  createdAt: string
  tags?: string[]
}

export async function listChangelog() {
  const result = await getData<PageResult<ChangelogEntry>>('/api/changelog')
  return result.items
}
