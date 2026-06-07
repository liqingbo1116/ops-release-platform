<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>用户管理</h1>
        <p>管理平台登录用户和角色绑定，当前为 mock 数据。</p>
      </div>
      <PermissionButton permission="user:write">新增用户</PermissionButton>
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="搜索用户、角色、状态" clearable />
      </div>
      <el-table :data="filteredRows" class="wide-table">
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
import { computed, ref } from 'vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { mockData } from '@/api/mockData'

const keyword = ref('')
const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return mockData.users
  return mockData.users.filter((item) => `${item.username} ${item.displayName} ${item.roles.join(' ')} ${item.status}`.toLowerCase().includes(q))
})
</script>
