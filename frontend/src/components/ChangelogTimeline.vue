<template>
  <div class="changelog-list">
    <el-card v-for="entry in items" :key="entry.id" shadow="never" class="changelog-card">
      <div class="changelog-head">
        <div>
          <div class="changelog-version">{{ entry.title }}</div>
          <div class="changelog-meta">{{ entry.createdAt }} / {{ entry.author || '未知' }}</div>
        </div>
        <el-tag :type="typeMap[entry.type] ?? 'info'" round>{{ labelMap[entry.type] ?? entry.type }}</el-tag>
      </div>

      <div v-if="entry.description" class="changelog-section">
        {{ entry.description }}
      </div>
      <div v-if="entry.tags?.length" class="changelog-tags">
        <el-tag v-for="tag in entry.tags" :key="tag" size="small" effect="plain">{{ tag }}</el-tag>
      </div>
    </el-card>
    <el-empty v-if="!items.length" description="暂无更新日志" />
  </div>
</template>

<script setup lang="ts">
import type { ChangelogEntry } from '@/api/changelog'

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

<style scoped>
.changelog-list {
  display: grid;
  gap: 12px;
}

.changelog-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.changelog-version {
  font-weight: 700;
}

.changelog-meta {
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.changelog-section {
  margin-top: 12px;
  color: var(--el-text-color-regular);
}

.changelog-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 12px;
}
</style>
