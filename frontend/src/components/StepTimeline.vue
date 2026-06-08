<template>
  <div class="steps">
    <div
      v-for="(step, index) in steps"
      :key="`${step.name}-${index}`"
      :class="['step-row', { active: isActiveStep(step) }]"
    >
      <div :class="['step-index', { active: isActiveStep(step) }]">{{ 'order' in step ? step.order : index + 1 }}</div>
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

const props = defineProps<{
  steps: Step[]
  activeStepName?: string
}>()

function isActiveStep(step: Step) {
  return Boolean(props.activeStepName && step.name === props.activeStepName)
}
</script>
