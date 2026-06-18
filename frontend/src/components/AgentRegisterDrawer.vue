<template>
  <el-drawer v-model="visible" title="注册 Agent" size="440px">
    <div class="drawer-stack">
      <el-form label-position="top">
        <el-form-item label="Agent ID">
          <el-input v-model="agentId" :disabled="loading" placeholder="agent-<project>-<env>" />
        </el-form-item>
        <el-form-item label="绑定环境">
          <el-select v-model="environmentId" style="width: 100%" :disabled="loading">
            <el-option v-for="env in environments" :key="env.id" :label="env.name" :value="env.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="注册 Token">
          <el-input :model-value="token" readonly placeholder="选择环境后生成 Token" />
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
      <el-button :loading="loading" :disabled="!environmentId" @click="generateToken">重新生成 Token</el-button>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { createAgentRegisterToken } from '@/api/agents'
import { formatDateTime } from '@/utils/format'

type Environment = {
  id: string
  name: string
}

const visible = defineModel<boolean>('visible', { required: true })
const props = defineProps<{
  environments: Environment[]
}>()

const environmentId = ref('')
const agentId = ref('')
const token = ref('')
const expiresAt = ref('')
const installCommand = ref('')
const loading = ref(false)

const expiresAtText = computed(() => (expiresAt.value ? formatDateTime(expiresAt.value) : ''))

watch(
  () => props.environments,
  (items) => {
    if (!environmentId.value && items.length > 0) {
      environmentId.value = items[0].id
      agentId.value = defaultAgentId(items[0].id)
    }
  },
  { immediate: true },
)

watch(environmentId, () => {
  agentId.value = defaultAgentId(environmentId.value)
  token.value = ''
  expiresAt.value = ''
  installCommand.value = ''
})

async function generateToken() {
  if (!environmentId.value || !agentId.value.trim()) {
    ElMessage.warning('请完整填写 Agent ID 和绑定环境')
    return
  }
  loading.value = true
  try {
    const result = await createAgentRegisterToken(environmentId.value, agentId.value.trim())
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

function defaultAgentId(id: string) {
  return id ? `agent-${id.replace(/^env-/, '')}` : ''
}
</script>
