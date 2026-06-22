<template>
  <el-drawer v-model="visible" title="注册 Agent" size="440px">
    <div class="drawer-stack">
      <el-form label-position="top">
        <el-form-item label="Agent 标识">
          <el-input v-model="agentId" :disabled="loading" placeholder="可不填，平台会自动生成" />
        </el-form-item>
        <el-form-item label="平台地址">
          <el-input :model-value="platformUrl" readonly placeholder="生成后显示" />
        </el-form-item>
        <el-form-item label="一次性注册密钥">
          <el-input :model-value="token" readonly placeholder="生成后显示一次性注册密钥" />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-input :model-value="expiresAtText" readonly placeholder="生成后显示" />
        </el-form-item>
        <el-form-item label="Agent 配置">
          <el-input
            :model-value="configText"
            type="textarea"
            :rows="12"
            readonly
            placeholder="生成后显示"
          />
        </el-form-item>
      </el-form>
      <el-alert
        v-if="notice"
        :type="noticeType"
        :closable="false"
        :title="notice"
      />
      <div class="drawer-actions">
        <el-button :loading="loading" type="primary" @click="generateToken">生成注册密钥</el-button>
        <el-button :disabled="!configText" @click="copyConfigText">复制配置</el-button>
      </div>
      <el-alert
        type="info"
        :closable="false"
        title="把生成的配置保存为项目环境机器上的 agent.conf。Agent 首次接入后会自动写回运行令牌，并显示为待认领。"
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
const configText = ref('')
const loading = ref(false)
const notice = ref('')
const noticeType = ref<'success' | 'warning' | 'error'>('warning')

const expiresAtText = computed(() => (expiresAt.value ? formatDateTime(expiresAt.value) : ''))

async function generateToken() {
  loading.value = true
  notice.value = ''
  try {
    const result = await createAgentRegisterToken(agentId.value.trim())
    agentId.value = result.agentId
    platformUrl.value = result.platformUrl
    token.value = result.token
    expiresAt.value = result.expiresAt
    configText.value = result.configText || result.installCommand || ''
    noticeType.value = 'success'
    notice.value = '注册密钥已生成，请复制配置并保存为项目环境机器上的 agent.conf。'
    ElMessage.success('注册密钥已生成')
  } catch (error) {
    noticeType.value = 'error'
    notice.value = error instanceof Error ? error.message : '注册密钥生成失败'
    ElMessage.error(notice.value)
  } finally {
    loading.value = false
  }
}

async function copyConfigText() {
  if (!configText.value) return
  try {
    await navigator.clipboard.writeText(configText.value)
    ElMessage.success('配置已复制')
  } catch {
    ElMessage.warning('复制失败，请手动复制')
  }
}
</script>

<style scoped>
.drawer-actions {
  display: flex;
  gap: 8px;
}
</style>
