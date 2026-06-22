<template>
  <el-drawer v-model="visible" title="注册 Agent" size="440px">
    <div class="drawer-stack">
      <el-form label-position="top">
        <el-form-item label="Agent ID">
          <el-input v-model="agentId" :disabled="loading" placeholder="agent-<project>-<env>" />
        </el-form-item>
        <el-form-item label="平台地址">
          <el-input :model-value="platformUrl" readonly placeholder="生成后显示" />
        </el-form-item>
        <el-form-item label="注册 Token">
          <el-input :model-value="token" readonly placeholder="生成后显示一次性 Token" />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-input :model-value="expiresAtText" readonly placeholder="生成后显示" />
        </el-form-item>
        <el-form-item label="安装指令">
          <el-input
            :model-value="installCommand"
            type="textarea"
            :rows="4"
            readonly
            placeholder="生成后显示"
          />
        </el-form-item>
      </el-form>
      <el-button type="primary" :disabled="!installCommand" @click="copyInstallCommand">复制注册指令</el-button>
      <el-button :loading="loading" @click="generateToken">生成 Token</el-button>
      <el-alert
        type="info"
        :closable="false"
        title="Agent 可以先注册为待认领；待认领 Agent 只展示在线状态，不能执行发布或部署任务。"
      />
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { createAgentRegisterToken } from '@/api/agents'
import { formatDateTime } from '@/utils/format'

const visible = defineModel<boolean>('visible', { required: true })

const agentId = ref('')
const token = ref('')
const platformUrl = ref('')
const expiresAt = ref('')
const installCommand = ref('')
const loading = ref(false)

const expiresAtText = computed(() => (expiresAt.value ? formatDateTime(expiresAt.value) : ''))

async function generateToken() {
  if (!agentId.value.trim()) {
    ElMessage.warning('请填写 Agent ID')
    return
  }
  loading.value = true
  try {
    const result = await createAgentRegisterToken(agentId.value.trim())
    platformUrl.value = result.platformUrl
    token.value = result.token
    expiresAt.value = result.expiresAt
    installCommand.value = result.installCommand
    ElMessage.success('注册 Token 已生成')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '注册 Token 生成失败')
  } finally {
    loading.value = false
  }
}

async function copyInstallCommand() {
  if (!installCommand.value) return
  try {
    await navigator.clipboard.writeText(installCommand.value)
    ElMessage.success('注册指令已复制')
  } catch {
    ElMessage.warning('复制失败，请手动复制')
  }
}
</script>
