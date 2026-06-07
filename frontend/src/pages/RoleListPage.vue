<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>角色管理</h1>
        <p>维护平台角色和权限集合，当前不接真实身份源。</p>
      </div>
      <PermissionButton permission="user:write">新增角色</PermissionButton>
    </div>

    <el-card shadow="never">
      <el-table :data="mockData.roles" class="wide-table">
        <el-table-column prop="code" label="角色编码" min-width="160" />
        <el-table-column prop="name" label="角色名称" min-width="140" />
        <el-table-column prop="description" label="说明" min-width="280" />
        <el-table-column label="权限" min-width="260">
          <template #default="{ row }">{{ row.permissions.join(' / ') }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import PermissionButton from '@/components/PermissionButton.vue'
import { mockData } from '@/api/mockData'
</script>
