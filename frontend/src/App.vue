<template>
  <el-config-provider>
    <router-view v-if="route.meta.bare" />
    <AppLayout v-else />
  </el-config-provider>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import AppLayout from '@/components/AppLayout.vue'

const route = useRoute()
</script>
