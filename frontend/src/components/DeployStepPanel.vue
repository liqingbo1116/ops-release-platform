<template>
  <el-card shadow="never">
    <template #header>
      <div class="panel-head">
        <strong>{{ title }}</strong>
        <StatusTag :status="status" />
      </div>
    </template>
    <StepTimeline :steps="steps" :active-step-name="activeStepName" />
  </el-card>
</template>

<script setup lang="ts">
import StatusTag from './StatusTag.vue'
import StepTimeline from './StepTimeline.vue'

type Step = {
  name: string
  status: string
  message?: string
  order?: number
  type?: string
}

defineProps<{
  title: string
  status: string
  steps: Step[]
  activeStepName?: string
}>()
</script>
