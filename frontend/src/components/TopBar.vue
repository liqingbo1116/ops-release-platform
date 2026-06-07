<template>
  <header class="topbar">
    <div class="crumb">运维发布交付平台 / <strong>{{ currentTitle }}</strong></div>
    <div class="top-actions">
      <el-select model-value="local-prod" style="width: 260px">
        <el-option label="本地生产环境 / local-prod" value="local-prod" />
        <el-option label="项目 X 生产环境 / project-x-prod" value="project-x-prod" />
      </el-select>
      <el-button @click="$router.push('/agents')">Agent 状态</el-button>
      <el-button type="primary" @click="$router.push('/releases/create')">新建发布单</el-button>
      <el-dropdown @command="handleCommand">
        <el-button>
          {{ authStore.user?.displayName ?? '未登录' }} / {{ authStore.user?.roles[0] ?? '-' }}
        </el-button>
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
