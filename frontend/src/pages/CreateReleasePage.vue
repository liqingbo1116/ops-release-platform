<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>创建发布单</h1>
        <p>强化批量服务选择、风险确认、自动回滚与跳过异常 workload 策略。</p>
      </div>
      <el-button type="primary" :loading="submitting" @click="submitRelease">{{ submitText }}</el-button>
    </div>

    <div class="two-col">
      <el-card shadow="never">
        <template #header><div class="panel-head"><strong>{{ configTitle }}</strong><el-tag>{{ configTag }}</el-tag></div></template>
        <el-form label-position="top" class="form-grid">
          <el-form-item label="发版方式">
            <el-radio-group v-model="releaseMode">
              <el-radio-button label="SERVICE_RELEASE">服务发版</el-radio-button>
              <el-radio-button label="SERVICE_DEPLOYMENT">服务部署</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="目标环境"><el-select model-value="env-project-x-prod"><el-option label="项目 X 生产 / project-x-prod" value="env-project-x-prod" /></el-select></el-form-item>
          <el-form-item v-if="releaseMode === 'SERVICE_RELEASE'" label="发版来源">
            <el-radio-group v-model="releaseSource">
              <el-radio-button label="JENKINS_JOB">Jenkins Job</el-radio-button>
              <el-radio-button label="LOCAL_HARBOR_IMAGE">本地 Harbor 镜像</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="releaseMode === 'SERVICE_RELEASE' && releaseSource === 'JENKINS_JOB'" label="Jenkins job">
            <el-select v-model="jenkinsJob">
              <el-option label="project-x-service-release" value="project-x-service-release" />
              <el-option label="project-x-image-tag-release" value="project-x-image-tag-release" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="releaseMode === 'SERVICE_RELEASE' && releaseSource === 'LOCAL_HARBOR_IMAGE'" label="本地 Harbor 镜像 tag">
            <el-select v-model="imageTag">
              <el-option label="harbor.local/project-x/user-service:20260607-a1b2c3" value="20260607-a1b2c3" />
              <el-option label="harbor.local/project-x/user-service:20260606-111aaa" value="20260606-111aaa" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="releaseMode === 'SERVICE_DEPLOYMENT'" label="来源基线 / 生产环境"><el-select model-value="BL-20260607-0001"><el-option label="BL-20260607-0001 / local-prod" value="BL-20260607-0001" /></el-select></el-form-item>
          <el-form-item label="执行 Agent"><el-select model-value="agent-project-x"><el-option label="agent-project-x / 在线" value="agent-project-x" /></el-select></el-form-item>
        </el-form>
      </el-card>

      <ReleaseRiskPanel v-model:options="options" :selected-count="selectedIds.length" />
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <strong>{{ selectionTitle }} <span class="mono">{{ selectedIds.length }}</span> 个服务</strong>
          <el-input v-model="keyword" placeholder="搜索服务、namespace、tag" clearable />
        </div>
        <div class="top-actions">
          <el-button @click="selectPublishable">{{ selectAllText }}</el-button>
          <el-button @click="selectedIds = []">清空选择</el-button>
        </div>
      </div>
      <ServiceDiffTable v-model:selected-ids="selectedIds" :items="filteredItems" />
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import ReleaseRiskPanel from '@/components/ReleaseRiskPanel.vue'
import ServiceDiffTable from '@/components/ServiceDiffTable.vue'
import { createDeployTask } from '@/api/deployTasks'
import { createRelease } from '@/api/releases'
import { mockData } from '@/api/mockData'

const router = useRouter()
const keyword = ref('')
const releaseMode = ref<'SERVICE_RELEASE' | 'SERVICE_DEPLOYMENT'>('SERVICE_RELEASE')
const releaseSource = ref<'JENKINS_JOB' | 'LOCAL_HARBOR_IMAGE'>('JENKINS_JOB')
const jenkinsJob = ref('project-x-service-release')
const imageTag = ref('20260607-a1b2c3')
const selectedIds = ref<string[]>([])
const submitting = ref(false)
const options = ref({
  autoRollback: true,
  skipWorkloadError: true,
  refreshTargetRuntime: true,
  auditLog: true,
})

const candidateItems = computed(() => {
  return mockData.diffResult.items.filter((item) => {
    if (releaseMode.value === 'SERVICE_DEPLOYMENT') {
      return item.diffStatus === 'MISSING_IN_TARGET'
    }
    return item.diffStatus !== 'MISSING_IN_TARGET'
  })
})

const filteredItems = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return candidateItems.value
  return candidateItems.value.filter((item) =>
    `${item.serviceName} ${item.namespace} ${item.sourceTag} ${item.targetTag ?? ''}`.toLowerCase().includes(q),
  )
})
const configTitle = computed(() => (releaseMode.value === 'SERVICE_DEPLOYMENT' ? '服务部署配置' : '服务发版配置'))
const configTag = computed(() => (releaseMode.value === 'SERVICE_DEPLOYMENT' ? '目标缺失服务' : '目标已有服务'))
const selectionTitle = computed(() => (releaseMode.value === 'SERVICE_DEPLOYMENT' ? '待部署' : '待发版'))
const selectAllText = computed(() => (releaseMode.value === 'SERVICE_DEPLOYMENT' ? '选择全部待部署' : '选择全部可发版'))
const submitText = computed(() => (releaseMode.value === 'SERVICE_DEPLOYMENT' ? '创建服务部署任务' : '提交服务发版'))

function selectPublishable() {
  selectedIds.value = filteredItems.value.filter((item) => item.publishable).map((item) => item.serviceId)
}

watch(releaseMode, () => {
  keyword.value = ''
  selectPublishable()
}, { immediate: true })

async function submitRelease() {
  submitting.value = true
  try {
    if (releaseMode.value === 'SERVICE_DEPLOYMENT') {
      const result = await createDeployTask({
        type: 'SERVICE_DEPLOYMENT',
        sourceBaselineId: mockData.diffResult.sourceBaselineId,
        targetEnvironmentId: mockData.diffResult.targetEnvironmentId,
        agentId: 'agent-project-x',
        serviceIds: selectedIds.value,
        options: {
          syncImage: true,
          createWorkload: true,
          healthCheck: true,
        },
      })
      ElMessage.success('服务部署任务已创建')
      router.push({ path: `/deploy-tasks/${result.id}`, query: { agentTaskId: result.id } })
      return
    }

    const result = await createRelease({
      type: 'SERVICE_RELEASE',
      targetEnvironmentId: mockData.diffResult.targetEnvironmentId,
      agentId: 'agent-project-x',
      serviceIds: selectedIds.value,
      releaseSource: releaseSource.value,
      image: releaseSource.value === 'LOCAL_HARBOR_IMAGE' ? {
        repository: 'harbor.local/project-x/user-service',
        tag: imageTag.value,
        digest: `sha256:mock-${imageTag.value}`,
      } : undefined,
      jenkins: releaseSource.value === 'JENKINS_JOB' ? {
        jobName: jenkinsJob.value,
        branch: 'main',
        parameters: {
          SERVICE_COUNT: String(selectedIds.value.length),
          TARGET_ENV: mockData.diffResult.targetEnvironmentId,
          RELEASE_SOURCE: releaseSource.value,
          IMAGE_TAG: imageTag.value,
        },
      } : undefined,
      options: options.value,
    })
    ElMessage.success('服务发版已提交 Jenkins')
    router.push({ path: `/releases/${result.id}`, query: { agentTaskId: result.agentTaskId ?? result.id } })
  } finally {
    submitting.value = false
  }
}
</script>
