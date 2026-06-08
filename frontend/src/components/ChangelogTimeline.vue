<template>
  <div class="changelog-list">
    <el-card v-for="entry in items" :key="entry.id" shadow="never" class="changelog-card">
      <div class="changelog-head">
        <div>
          <div class="changelog-version">{{ entry.version }} · {{ entry.title }}</div>
          <div class="changelog-meta">{{ entry.releasedAt }} / {{ entry.operator }}</div>
        </div>
        <el-tag :type="typeMap[entry.type] ?? 'info'" round>{{ labelMap[entry.type] ?? entry.type }}</el-tag>
      </div>

      <div class="changelog-section">
        <strong>新增功能</strong>
        <ul>
          <li v-for="item in entry.features" :key="item">{{ item }}</li>
        </ul>
      </div>
      <div v-if="entry.fixes.length" class="changelog-section">
        <strong>修复问题</strong>
        <ul>
          <li v-for="item in entry.fixes" :key="item">{{ item }}</li>
        </ul>
      </div>
      <div v-if="entry.knownIssues.length" class="changelog-section">
        <strong>已知问题</strong>
        <ul>
          <li v-for="item in entry.knownIssues" :key="item">{{ item }}</li>
        </ul>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { changelogMockData } from '@/api/mockData/changelog'

type ChangelogEntry = (typeof changelogMockData.changelog)[number]

defineProps<{
  items: ChangelogEntry[]
}>()

const typeMap: Record<string, 'success' | 'warning' | 'danger' | 'info' | 'primary'> = {
  FEATURE: 'primary',
  FIX: 'danger',
  IMPROVEMENT: 'success',
  SECURITY: 'warning',
}

const labelMap: Record<string, string> = {
  FEATURE: '功能',
  FIX: '修复',
  IMPROVEMENT: '优化',
  SECURITY: '安全',
}
</script>
