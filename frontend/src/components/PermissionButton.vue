<template>
  <el-button :disabled="!allowed" :type="type">
    <slot />
  </el-button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from '@/stores/authStore'

const props = withDefaults(
  defineProps<{
    permission: string
    type?: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  }>(),
  {
    type: 'primary',
  },
)

const authStore = useAuthStore()
const allowed = computed(() => authStore.hasPermission(props.permission))
</script>
