<template>
  <el-card class="metric-card" shadow="never">
    <div class="metric-label">{{ label }}</div>
    <div class="metric-value">{{ value }}</div>
    <div v-if="foot" :class="['metric-foot', tone]">{{ foot }}</div>
  </el-card>
</template>

<script setup lang="ts">
defineProps<{
  label: string
  value: string | number
  foot?: string
  tone?: 'good' | 'warn' | 'bad'
}>()
</script>
