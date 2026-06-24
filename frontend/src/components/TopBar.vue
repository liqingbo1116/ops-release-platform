<template>
  <header class="topbar">
    <div class="crumb">运维发布交付平台 / <strong>{{ currentTitle }}</strong></div>
    <div class="top-actions">
      <el-dropdown @command="handleCommand">
        <button class="top-user-trigger" type="button">
          <span class="top-user-name">{{ authStore.user?.displayName ?? '未登录' }}</span>
          <span class="top-user-role">{{ authStore.user?.roles[0] ?? '-' }}</span>
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="logout">退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const currentTitle = computed(() => String(route.meta.title ?? '首页工作台'))

authStore.loadCurrentUser()

async function handleCommand(command: string) {
  if (command === 'logout') {
    await authStore.logout()
    await router.push('/login')
  }
}
</script>
