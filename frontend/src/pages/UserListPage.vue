<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>用户管理</h1>
        <p>管理平台登录用户和角色绑定。</p>
      </div>
      <PermissionButton permission="user:write">新增用户</PermissionButton>
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="搜索用户、角色、状态" clearable />
      </div>
      <el-table v-loading="loading" :data="filteredRows" class="wide-table">
        <el-table-column prop="username" label="用户名" min-width="140" />
        <el-table-column prop="displayName" label="显示名称" min-width="140" />
        <el-table-column label="角色" min-width="180">
          <template #default="{ row }">{{ row.roles.join(' / ') }}</template>
        </el-table-column>
        <el-table-column label="状态" min-width="110">
          <template #default="{ row }">
            <el-tag :type="row.status === 'ENABLED' ? 'success' : 'info'" round>{{ row.status === 'ENABLED' ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="lastLoginAt" label="最近登录" min-width="180" />
      </el-table>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { listUsers, type UserInfo } from '@/api/users'

const keyword = ref('')
const loading = ref(false)
const rows = ref<UserInfo[]>([])
const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return rows.value
  return rows.value.filter((item) => `${item.username} ${item.displayName} ${item.roles.join(' ')} ${item.status}`.toLowerCase().includes(q))
})

async function loadRows() {
  loading.value = true
  try {
    rows.value = await listUsers()
  } catch {
    ElMessage.error('加载用户失败')
    rows.value = []
  } finally {
    loading.value = false
  }
}

onMounted(loadRows)
</script>
