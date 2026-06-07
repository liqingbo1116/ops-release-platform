<template>
  <el-drawer v-model="visible" title="注册 Agent" size="440px">
    <div class="drawer-stack">
      <el-form label-position="top">
        <el-form-item label="绑定环境">
          <el-select v-model="environmentId" style="width: 100%">
            <el-option v-for="env in environments" :key="env.id" :label="env.name" :value="env.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="注册 Token">
          <el-input model-value="agt_7f92c1b8_20260607" readonly />
        </el-form-item>
        <el-form-item label="安装指令">
          <el-input
            :model-value="installCommand"
            type="textarea"
            :rows="4"
            readonly
          />
        </el-form-item>
      </el-form>
      <el-button type="primary">复制注册指令</el-button>
      <el-button>重新生成 Token</el-button>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

type Environment = {
  id: string
  name: string
}

const visible = defineModel<boolean>('visible', { required: true })
defineProps<{
  environments: Environment[]
}>()

const environmentId = ref('env-project-x-prod')
const installCommand = computed(
  () =>
    'curl -fsSL https://platform.local/agent/install.sh | bash -s -- --token agt_7f92c1b8_20260607 --server https://platform.local',
)
</script>
