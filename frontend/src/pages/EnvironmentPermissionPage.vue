<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>环境权限</h1>
        <p>控制角色可访问的环境范围和动作。</p>
      </div>
      <PermissionButton permission="user:write">保存权限</PermissionButton>
    </div>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="rows" class="wide-table">
        <el-table-column prop="environmentName" label="环境" min-width="170" />
        <el-table-column prop="roleName" label="角色" min-width="150" />
        <el-table-column label="动作" min-width="240">
          <template #default="{ row }">{{ row.actions.join(' / ') }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { onMounted, ref } from 'vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { listPermissions, type EnvironmentPermission } from '@/api/users'

const loading = ref(false)
const rows = ref<EnvironmentPermission[]>([])

async function loadRows() {
  loading.value = true
  try {
    rows.value = await listPermissions()
  } catch {
    ElMessage.error('加载环境权限失败')
    rows.value = []
  } finally {
    loading.value = false
  }
}

onMounted(loadRows)
</script>
