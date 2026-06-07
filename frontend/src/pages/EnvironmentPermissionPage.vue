<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>环境权限</h1>
        <p>控制角色可访问的环境范围和动作，当前为 mock 权限矩阵。</p>
      </div>
      <PermissionButton permission="user:write">保存权限</PermissionButton>
    </div>

    <el-card shadow="never">
      <el-table :data="mockData.permissions" class="wide-table">
        <el-table-column prop="environmentName" label="环境" min-width="170" />
        <el-table-column prop="roleCode" label="角色" min-width="150" />
        <el-table-column prop="scope" label="范围" min-width="120" />
        <el-table-column label="动作" min-width="240">
          <template #default="{ row }">{{ row.actions.join(' / ') }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import PermissionButton from '@/components/PermissionButton.vue'
import { mockData } from '@/api/mockData'
</script>
