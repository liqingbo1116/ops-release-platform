<template>
  <div class="steps">
    <div v-for="(step, index) in steps" :key="`${step.name}-${index}`" class="step-row">
      <div class="step-index">{{ 'order' in step ? step.order : index + 1 }}</div>
      <div class="step-main">
        <strong>{{ step.name }}</strong>
        <span v-if="'message' in step && step.message">{{ step.message }}</span>
        <span v-if="'type' in step">{{ step.type }}</span>
      </div>
      <StatusTag :status="step.status" />
    </div>
  </div>
</template>

<script setup lang="ts">
import StatusTag from './StatusTag.vue'

type Step = {
  name: string
  status: string
  message?: string
  order?: number
  type?: string
}

defineProps<{
  steps: Step[]
}>()
</script>
