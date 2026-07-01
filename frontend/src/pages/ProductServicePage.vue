<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>服务管理</h1>
        <p>{{ productTitle }}</p>
      </div>
    </div>

    <el-alert v-if="errorMessage" class="service-alert" type="warning" :closable="false" :title="errorMessage" />

    <div class="service-control-bar">
      <div class="product-switcher">
        <label>
          <span>项目</span>
          <el-select
            v-model="selectedProjectId"
            filterable
            clearable
            placeholder="选择项目"
            :disabled="projects.length === 0"
            @change="handleProjectChange"
          >
            <el-option
              v-for="item in projects"
              :key="item.id"
              :label="`${item.name} / ${item.code}`"
              :value="item.id"
            />
          </el-select>
        </label>
        <label>
          <span>产品</span>
          <el-select
            v-model="selectedProductId"
            filterable
            placeholder="选择产品"
            :disabled="filteredProductsForProject.length === 0"
            @change="handleProductChange"
          >
            <el-option
              v-for="item in filteredProductsForProject"
              :key="item.id"
              :label="`${item.name} / ${item.code}`"
              :value="item.id"
            />
          </el-select>
        </label>
      </div>
      <div class="service-summary">
        <span>已纳管 <strong>{{ managedServices.length }}</strong></span>
        <span>可纳管 <strong>{{ unmanagedServices.length }}</strong></span>
        <span class="pipeline-summary">
          Jenkins view
          <strong>{{ productJenkinsViewText }}</strong>
        </span>
      </div>
    </div>

    <section class="service-list-panel">
      <div class="panel-head">
        <div class="panel-title">
          <strong>已纳管服务</strong>
          <span>已纳管 {{ managedServices.length }} / 可纳管 {{ unmanagedServices.length }}</span>
        </div>
        <div class="service-actions">
          <el-tag size="small" effect="plain">已选择 {{ selectedManagedServices.length }}</el-tag>
          <el-button size="small" :loading="syncingManagedServices" @click="refreshManagedServices">刷新</el-button>
          <el-button type="primary" size="small" plain @click="openAdoptDialog">纳管服务</el-button>
          <el-button
            type="danger"
            size="small"
            plain
            :disabled="selectedManagedServices.length === 0"
            :loading="removing"
            @click="removeSelectedManagedServices"
          >
            移除所选
          </el-button>
        </div>
      </div>
      <div class="service-filter">
        <el-input v-model="serviceFilters.keyword" clearable placeholder="搜索服务、镜像、命名空间" />
        <el-select v-model="serviceFilters.namespace" clearable placeholder="命名空间">
          <el-option v-for="item in managedNamespaces" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select v-model="serviceFilters.workloadType" clearable placeholder="工作负载">
          <el-option v-for="item in managedWorkloadTypes" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select v-model="serviceFilters.imageSource" clearable placeholder="镜像来源">
          <el-option label="私有镜像" value="PRIVATE" />
          <el-option label="私有项目未纳管" value="UNMATCHED_PRIVATE" />
          <el-option label="公共/外部镜像" value="EXTERNAL" />
        </el-select>
        <el-select v-model="serviceFilters.pipeline" clearable placeholder="Pipeline">
          <el-option label="已绑定" value="BOUND" />
          <el-option label="未绑定" value="UNBOUND" />
        </el-select>
      </div>
      <div v-if="managedServices.length > 0" class="registry-panel">
        <div>
          <strong>私有镜像 registry</strong>
          <span>{{ registryPanelText }}</span>
        </div>
        <div v-if="managedRegistryConfirmed" class="registry-confirmed">
          <el-tag size="small" type="success" effect="light">{{ managedRegistryHost }}</el-tag>
        </div>
        <div v-else-if="registryCandidates.length > 0" class="registry-confirm">
          <el-select v-model="selectedRegistryHost" size="small" placeholder="选择 registry" class="registry-select">
            <el-option v-for="host in registryCandidates" :key="host" :label="host" :value="host" />
          </el-select>
          <el-button
            type="primary"
            size="small"
            :disabled="!selectedRegistryHost"
            :loading="confirmingRegistry"
            @click="confirmManagedRegistry"
          >
            确认
          </el-button>
        </div>
      </div>
      <el-empty v-if="!loading && managedServices.length === 0" description="暂无已纳管服务" />
      <el-table
        v-else
        ref="managedTableRef"
        v-loading="loading || syncingManagedServices"
        :data="filteredManagedServices"
        class="service-table"
        @selection-change="handleManagedSelectionChange"
      >
        <el-table-column type="selection" width="48" />
        <el-table-column label="服务" min-width="240">
          <template #default="{ row }">
            <div class="service-name-cell">
              <el-button text type="primary" class="service-link" @click="openServiceDetail(row)">
                {{ row.name }}
              </el-button>
              <span>{{ row.namespace }} / {{ row.workloadType }} / {{ row.workloadName }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="容器" width="150">
          <template #default="{ row }">
            <el-tag size="small" :type="containerTagType(row.containerType)" effect="light">
              {{ containerTypeLabel(row.containerType) }}
            </el-tag>
            <div class="container-name">{{ row.containerName }}</div>
          </template>
        </el-table-column>
        <el-table-column label="镜像" min-width="300">
          <template #default="{ row }">
            <div class="image-cell">
              <span>{{ row.image }}</span>
              <div class="image-meta">
                <el-tooltip :content="imageSourceTip(row, 'managed')" placement="top">
                  <el-tag size="small" :type="imageSourceTagType(row.imageSource)" effect="light">
                    {{ imageSourceLabel(row, 'managed') }}
                  </el-tag>
                </el-tooltip>
                <span>{{ row.imageProject || '-' }} / {{ row.imageTag || '无 tag' }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
	        <el-table-column label="Pipeline" min-width="190">
	          <template #default="{ row }">
	            <div class="pipeline-cell">
	              <el-tooltip :content="pipelineBindingTip(row)" placement="top">
	                <el-tag size="small" :type="pipelineBindingTagType(row)" effect="light">
	                  {{ pipelineBindingLabel(row) }}
	                </el-tag>
	              </el-tooltip>
	              <span :class="{ 'is-invalid': pipelineBindingInvalid(row) }">{{ pipelineName(row) || '首次发版前选择 Pipeline' }}</span>
	              <el-button text type="primary" size="small" @click="openPipelineDialog(row)">
	                {{ pipelineHasStoredBinding(row) ? '更换' : '绑定' }}
	              </el-button>
	            </div>
	          </template>
	        </el-table-column>
        <el-table-column label="副本" width="90">
          <template #default="{ row }">{{ row.readyReplicas }}/{{ row.replicas }}</template>
        </el-table-column>
	        <el-table-column label="操作" fixed="right" width="140">
	          <template #default="{ row }">
	            <el-button text type="primary" size="small" @click="openServiceDetail(row)">详情</el-button>
	            <el-button text type="primary" size="small" @click="openReleaseDialog(row)">发版</el-button>
	          </template>
	        </el-table-column>
      </el-table>
    </section>

    <el-drawer v-model="serviceDetailVisible" title="服务发版" size="760px" class="service-detail-drawer">
      <template v-if="activeService">
        <div class="detail-head">
          <div>
            <strong>{{ activeService.name }}</strong>
            <span>{{ product?.projectName || '未绑定项目' }} / {{ product?.name || '-' }}</span>
	          </div>
	          <div class="detail-actions">
	            <el-button size="small" @click="openPipelineDialog(activeService)">
	              {{ pipelineHasStoredBinding(activeService) ? '更换 Pipeline' : '绑定 Pipeline' }}
	            </el-button>
	            <el-button type="primary" size="small" @click="openReleaseDialog(activeService)">发版</el-button>
	          </div>
	        </div>
        <div class="service-detail-grid">
          <div><span>命名空间</span><strong>{{ activeService.namespace || '-' }}</strong></div>
          <div><span>工作负载</span><strong>{{ activeService.workloadType }} / {{ activeService.workloadName }}</strong></div>
          <div><span>容器</span><strong>{{ containerTypeLabel(activeService.containerType) }} / {{ activeService.containerName }}</strong></div>
          <div><span>副本</span><strong>{{ activeService.readyReplicas }}/{{ activeService.replicas }}</strong></div>
          <div class="span-2"><span>镜像</span><strong>{{ activeService.image || '-' }}</strong></div>
	          <div>
	            <span>Pipeline</span>
	            <strong :class="{ 'is-invalid': pipelineBindingInvalid(activeService) }">{{ pipelineName(activeService) || '未绑定' }}</strong>
	          </div>
        </div>
        <div class="history-head">
          <strong>发版历史</strong>
          <el-button size="small" :loading="loadingServiceReleases" @click="loadServiceReleaseHistory">刷新</el-button>
        </div>
        <el-table
          v-loading="loadingServiceReleases"
          :data="serviceReleaseHistory"
          class="release-history-table"
          empty-text="暂无发版历史"
        >
          <el-table-column label="时间" width="156">
            <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
          </el-table-column>
          <el-table-column label="发版单" min-width="150">
            <template #default="{ row }">
              <router-link :to="`/releases/${row.id}`">{{ row.id }}</router-link>
            </template>
          </el-table-column>
          <el-table-column label="Jenkins" min-width="190">
            <template #default="{ row }">
              <div class="history-jenkins-cell">
                <el-tag size="small" :type="releaseStatusTagType(row.buildStatus)" effect="light">
                  {{ releaseStatusLabel(row.buildStatus) }}
                </el-tag>
                <a v-if="row.buildUrl" :href="row.buildUrl" target="_blank" rel="noreferrer">
                  {{ row.buildId || '构建日志' }}
                </a>
                <span v-else>{{ row.buildId || '-' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="目标镜像" min-width="260">
            <template #default="{ row }">
              <div class="history-image-cell">
                <span>{{ releaseImageVersion(row) }}</span>
                <small>{{ releaseImageTip(row) }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-tag size="small" :type="releaseStatusTagType(row.status)" effect="light">
                {{ releaseStatusLabel(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </el-drawer>

    <el-dialog v-model="adoptDialogVisible" title="纳管服务" width="980px">
      <div class="dialog-table-head">
        <span>发现 {{ discoveredServices.length }} 个容器服务，可选择多个服务纳入当前产品。</span>
        <strong>已选择 {{ selectedDiscoveredServices.length }} 个</strong>
      </div>
      <el-table
        ref="discoveredTableRef"
        v-loading="loading"
        :data="discoveredServices"
        class="service-table"
        @selection-change="handleDiscoveredSelectionChange"
      >
        <el-table-column type="selection" width="48" :selectable="selectableDiscoveredService" />
        <el-table-column label="服务" min-width="240">
          <template #default="{ row }">
            <div class="service-name-cell">
              <strong>{{ row.name }}</strong>
              <span>{{ row.namespace }} / {{ row.workloadType }} / {{ row.workloadName }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="容器" width="150">
          <template #default="{ row }">
            <el-tag size="small" :type="containerTagType(row.containerType)" effect="light">
              {{ containerTypeLabel(row.containerType) }}
            </el-tag>
            <div class="container-name">{{ row.containerName }}</div>
          </template>
        </el-table-column>
        <el-table-column label="镜像" min-width="360">
          <template #default="{ row }">
            <div class="image-cell">
              <span>{{ row.image }}</span>
              <div class="image-meta">
                <el-tooltip :content="imageSourceTip(row, 'discovered')" placement="top">
                  <el-tag size="small" :type="imageSourceTagType(row.imageSource)" effect="light">
                    {{ imageSourceLabel(row, 'discovered') }}
                  </el-tag>
                </el-tooltip>
                <span>{{ row.imageProject || '-' }} / {{ row.imageTag || '无 tag' }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.managed" size="small" type="success" effect="light">已纳管</el-tag>
            <el-tag v-else size="small" type="info" effect="light">可纳管</el-tag>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="adoptDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :disabled="selectedDiscoveredServices.length === 0"
          :loading="adopting"
          @click="adoptSelectedServices"
        >
          纳管所选服务
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="pipelineDialogVisible" title="绑定 Jenkins Pipeline" width="520px">
      <div class="dialog-context">
        <div>
          <span>当前产品 Jenkins view</span>
          <strong>{{ productJenkinsViewText }}</strong>
        </div>
        <div>
          <span>可选 Pipeline</span>
          <strong>{{ availablePipelinesForActiveService.length }} / {{ jenkinsPipelines.length }}</strong>
        </div>
      </div>
      <el-form label-width="92px">
        <el-form-item label="服务">
          <span>{{ activeService?.name || '-' }}</span>
        </el-form-item>
        <el-form-item label="Pipeline">
          <el-select
            v-model="pipelineForm.pipelineKey"
            filterable
            placeholder="选择已发现 Pipeline"
            class="dialog-control"
            :disabled="availablePipelinesForActiveService.length === 0"
            :empty-values="[]"
            :value-on-clear="''"
          >
            <el-option
              v-for="pipeline in availablePipelinesForActiveService"
              :key="pipelineCandidateKey(pipeline)"
              :label="pipelineOptionLabel(pipeline)"
              :value="pipelineCandidateKey(pipeline)"
            />
          </el-select>
          <el-alert
            v-if="pipelineSelectNotice"
            class="inline-alert"
            type="warning"
            :closable="false"
            :title="pipelineSelectNotice"
          />
          <div class="form-tip">仅显示当前产品绑定 Jenkins view 下的 Pipeline，且已被其他服务绑定的 Pipeline 不可重复选择。</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pipelineDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="bindingPipeline" @click="submitPipelineBinding">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="releaseDialogVisible" title="服务发版" width="680px">
      <div class="release-summary">
        <div><span>产品</span><strong>{{ product?.name || '-' }}</strong></div>
        <div><span>服务</span><strong>{{ activeService?.name || '-' }}</strong></div>
        <div><span>Pipeline</span><strong>{{ activeService ? pipelineName(activeService) || '-' : '-' }}</strong></div>
        <div><span>当前镜像</span><strong>{{ activeService?.image || '-' }}</strong></div>
      </div>
      <el-radio-group v-model="releaseMode" class="release-mode-group">
        <el-radio-button label="JENKINS_JOB">基于 Jenkins 发版</el-radio-button>
        <el-radio-button label="LOCAL_HARBOR_IMAGE">基于镜像发版</el-radio-button>
      </el-radio-group>
      <el-alert
        v-if="releaseWarning"
        class="dialog-alert"
        type="warning"
        :closable="false"
        :title="releaseWarning"
      />
      <div v-if="releaseMode === 'JENKINS_JOB' && !releaseWarning" class="release-parameters">
        <div class="section-title">发版参数</div>
        <el-form label-width="120px">
          <el-form-item
            v-for="param in activePipelineParameters"
            :key="param.name"
            :label="param.name"
            :required="param.required"
          >
            <el-input
              v-model="releaseParameters[param.name]"
              :placeholder="pipelineParameterPlaceholder(param)"
              class="dialog-control"
            />
            <div v-if="param.description || param.required" class="form-tip">
              {{ pipelineParameterTip(param) }}
            </div>
          </el-form-item>
        </el-form>
        <div v-if="activePipelineParameters.length === 0" class="form-tip standalone-tip">
          当前 Pipeline 未发现参数，确认后将直接触发 Jenkins 构建。
        </div>
      </div>
      <div v-if="releaseMode === 'LOCAL_HARBOR_IMAGE' && !releaseWarning" class="release-parameters">
        <div class="section-title">选择镜像版本</div>
        <el-form label-width="120px">
          <el-form-item label="镜像仓库">
            <span class="repository-text">{{ activeReleaseSource?.imageRepository || activeService?.imageRepository || '-' }}</span>
          </el-form-item>
          <el-form-item label="镜像 tag" required>
            <el-select v-model="selectedImageTag" filterable placeholder="选择 Harbor 中已存在的镜像 tag" class="dialog-control">
              <el-option
                v-for="tag in activeImageTags"
                :key="`${tag.tag}-${tag.digest || ''}`"
                :label="imageTagOptionLabel(tag)"
                :value="tag.tag"
              />
            </el-select>
          </el-form-item>
        </el-form>
        <div class="form-tip standalone-tip">
          镜像列表来自当前产品绑定的 Harbor。选择后进入发版日志，服务当前镜像以实际 K8s/Agent 上报为准。
        </div>
      </div>
      <template #footer>
        <el-button @click="releaseDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creatingRelease" :disabled="Boolean(releaseWarning)" @click="submitRelease">
          触发并查看日志
        </el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type TableInstance } from 'element-plus'
import {
  adoptEnvironmentServices,
  bindEnvironmentServicePipeline,
  confirmEnvironmentServiceRegistry,
  listDiscoveredEnvironmentServices,
  listEnvironments,
  listEnvironmentServices,
  removeEnvironmentServices,
  syncEnvironmentServices,
  type DiscoveredProductService,
  type EnvironmentInfo,
  type ProductService,
} from '@/api/environments'
import { listAgents, type AgentInfo } from '@/api/agents'
import { listJenkinsInstances, type JenkinsInstance } from '@/api/integrationResources'
import { listProjects, type ProjectInfo } from '@/api/projects'
import {
  createRelease,
  listReleaseSources,
  listServiceReleases,
  type JenkinsPipeline,
  type ReleaseImageTag,
  type ReleaseOrder,
  type ReleaseSourceService,
} from '@/api/releases'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const syncingManagedServices = ref(false)
const adopting = ref(false)
const confirmingRegistry = ref(false)
const removing = ref(false)
const bindingPipeline = ref(false)
const creatingRelease = ref(false)
const releaseSourceLoading = ref(false)
const releaseSourceError = ref('')
const loadingServiceReleases = ref(false)
const errorMessage = ref('')
const products = ref<EnvironmentInfo[]>([])
const projects = ref<ProjectInfo[]>([])
const selectedProjectId = ref('')
const selectedProductId = ref('')
const product = ref<EnvironmentInfo | null>(null)
const managedServices = ref<ProductService[]>([])
const discoveredServices = ref<DiscoveredProductService[]>([])
const releaseSourceServices = ref<ReleaseSourceService[]>([])
const jenkinsPipelines = ref<JenkinsPipeline[]>([])
const jenkinsInstances = ref<JenkinsInstance[]>([])
const agents = ref<AgentInfo[]>([])
const selectedManagedServices = ref<ProductService[]>([])
const selectedDiscoveredServices = ref<DiscoveredProductService[]>([])
const selectedRegistryHost = ref('')
const managedTableRef = ref<TableInstance>()
const discoveredTableRef = ref<TableInstance>()
const adoptDialogVisible = ref(false)
const pipelineDialogVisible = ref(false)
const releaseDialogVisible = ref(false)
const serviceDetailVisible = ref(false)
const activeService = ref<ProductService | null>(null)
const serviceReleaseHistory = ref<ReleaseOrder[]>([])
const pipelineForm = ref({ pipelineKey: '' })
const releaseParameters = ref<Record<string, string>>({})
const releaseMode = ref<'JENKINS_JOB' | 'LOCAL_HARBOR_IMAGE'>('JENKINS_JOB')
const selectedImageTag = ref('')
const serviceFilters = ref({
  keyword: '',
  namespace: '',
  workloadType: '',
  imageSource: '',
  pipeline: '',
})
let managedServiceSyncTimer: ReturnType<typeof window.setInterval> | undefined
let managedServiceInitialSyncTimer: ReturnType<typeof window.setTimeout> | undefined
const MANAGED_SERVICE_SYNC_INTERVAL_MS = 30000

const standaloneMode = computed(() => route.name === 'services' || route.path === '/services')
const productId = computed(() => selectedProductId.value || (standaloneMode.value ? '' : String(route.params.id || '')))
const filteredProductsForProject = computed(() => {
  if (!selectedProjectId.value) return products.value
  return products.value.filter((item) => item.projectId === selectedProjectId.value)
})
const productTitle = computed(() => {
  if (!product.value) return products.value.length > 0 ? '请选择产品查看服务。' : '请先创建产品，再从探测结果中纳管服务。'
  const sourceText = product.value.type === 'LOCAL' ? '平台直连探测' : 'Agent 上报探测'
  const projectText = product.value.projectName || '未绑定项目'
  return `${projectText} / ${product.value.name} / ${product.value.code} / ${sourceText}`
})
const productJenkinsViewText = computed(() => {
  const viewNames = scopedValues(product.value, 'JENKINS')
  return viewNames.length > 0 ? viewNames.join('、') : product.value?.jenkinsView || '未绑定'
})
const productJenkinsIds = computed(() => {
  const ids = new Set<string>()
  if (product.value?.jenkinsId?.trim()) ids.add(product.value.jenkinsId.trim())
  for (const binding of product.value?.bindings ?? []) {
    if (binding.resourceType === 'JENKINS' && binding.resourceId?.trim()) ids.add(binding.resourceId.trim())
  }
  return [...ids]
})
const productJenkinsViewNames = computed(() => {
  const views = scopedValues(product.value, 'JENKINS')
  if (views.length > 0) return views
  return product.value?.jenkinsView ? [product.value.jenkinsView] : []
})
const unmanagedServices = computed(() => discoveredServices.value.filter((item) => !item.managed))
const releaseSourceByServiceId = computed(() => {
  const services = new Map<string, ReleaseSourceService>()
  for (const service of releaseSourceServices.value) {
    services.set(service.serviceId, service)
  }
  return services
})
const boundPipelineNames = computed(() => {
  const names = new Map<string, string>()
  for (const service of managedServices.value) {
    const name = pipelineName(service).trim()
    if (!name) continue
    const pipeline = findProductPipeline(service)
    if (pipeline) {
      names.set(pipelineCandidateKey(pipeline), service.id)
      if (pipeline.url) names.set(pipeline.url.trim(), service.id)
    }
    names.set(name, service.id)
  }
  return names
})
const availablePipelinesForActiveService = computed(() => {
  const activeServiceId = activeService.value?.id || ''
  return filterPipelinesForCurrentProduct(jenkinsPipelines.value).filter((pipeline) => {
    const boundServiceId = boundPipelineNames.value.get(pipelineCandidateKey(pipeline)) || boundPipelineNames.value.get(pipeline.name)
    return !boundServiceId || boundServiceId === activeServiceId
  })
})
const pipelineSelectNotice = computed(() => {
  if (!pipelineDialogVisible.value) return ''
  if (jenkinsPipelines.value.length === 0) {
    const jenkinsText = productJenkinsIds.value.length > 0 ? productJenkinsIds.value.join('、') : '未绑定 Jenkins'
    const viewText = productJenkinsViewNames.value.length > 0 ? productJenkinsViewNames.value.join('、') : '未绑定 view'
    const localPipelineCount = boundJenkinsInstancePipelineCount.value
    if (localPipelineCount > 0) {
      return `当前产品绑定的 Jenkins view 未匹配到 Pipeline。Jenkins：${jenkinsText}；view：${viewText}；基础资源中该 Jenkins 已发现 ${localPipelineCount} 个 Pipeline，请检查产品绑定的 view 名称是否与 Jenkins 视图一致。`
    }
    const statusText = productJenkinsProbeText.value
    return `当前产品绑定的 Jenkins view 未发现 Pipeline。Jenkins：${jenkinsText}；view：${viewText}。${statusText}`
  }
  if (availablePipelinesForActiveService.value.length === 0) return '当前 Jenkins view 下的 Pipeline 已全部绑定其他服务。'
  return ''
})
const activePipeline = computed(() => {
  return activeService.value ? findProductPipeline(activeService.value) : null
})
const activePipelineParametersUnknown = computed(() => Boolean(activePipeline.value && !Array.isArray(activePipeline.value.parameters)))
const activePipelineParameters = computed(() => activePipeline.value?.parameters ?? [])
const activeBranchParameter = computed(() =>
  activePipelineParameters.value.find((param) => isBranchParameterName(param.name)) ?? null,
)
const activeReleaseSource = computed(() => (activeService.value ? releaseSourceOf(activeService.value) : undefined))
const activeImageTags = computed(() => activeReleaseSource.value?.tags ?? [])
const managedNamespaces = computed(() => uniqueSorted(managedServices.value.map((item) => item.namespace)))
const managedWorkloadTypes = computed(() => uniqueSorted(managedServices.value.map((item) => item.workloadType)))
const boundJenkinsInstancePipelineCount = computed(() => {
  const ids = new Set(productJenkinsIds.value)
  return jenkinsInstances.value
    .filter((item) => ids.has(item.id))
    .reduce((total, item) => total + (item.pipelines?.length || item.jobs?.length || 0), 0)
})
const productJenkinsProbeText = computed(() => {
  const ids = new Set(productJenkinsIds.value)
  const instances = jenkinsInstances.value.filter((item) => ids.has(item.id))
  if (instances.length === 0) return '当前产品未匹配到 Jenkins 基础资源，请先检查产品绑定。'
  return instances
    .map((item) => {
      const status = item.status || 'UNKNOWN'
      const message = item.probeMessage || '无探测信息'
      const viewCount = item.views?.length || 0
      const pipelineCount = item.pipelines?.length || 0
      return `${item.name}：${status}，view ${viewCount} 个，Pipeline ${pipelineCount} 个，${message}`
    })
    .join('；')
})
const filteredManagedServices = computed(() => {
  const keyword = serviceFilters.value.keyword.trim().toLowerCase()
  return managedServices.value.filter((item) => {
    if (serviceFilters.value.namespace && item.namespace !== serviceFilters.value.namespace) return false
    if (serviceFilters.value.workloadType && item.workloadType !== serviceFilters.value.workloadType) return false
    if (serviceFilters.value.imageSource && item.imageSource !== serviceFilters.value.imageSource) return false
    if (serviceFilters.value.pipeline === 'BOUND' && !pipelineBound(item)) return false
    if (serviceFilters.value.pipeline === 'UNBOUND' && pipelineBound(item)) return false
    if (!keyword) return true
    return [item.name, item.namespace, item.workloadName, item.workloadType, item.containerName, item.image, pipelineName(item)]
      .join(' ')
      .toLowerCase()
      .includes(keyword)
  })
})
const activeAgent = computed(() => {
  if (product.value?.networkMode !== 'AGENT') return null
  return agents.value.find(
    (item) => item.environmentId === productId.value && item.claimStatus === 'CLAIMED' && item.status === 'ONLINE',
  ) ?? null
})
const releaseWarning = computed(() => {
  if (!activeService.value) return '请选择要发版的服务'
  if (product.value?.networkMode === 'AGENT' && !activeAgent.value) return '远程产品需要在线且已绑定当前产品的 Agent'
  if (releaseMode.value === 'LOCAL_HARBOR_IMAGE') {
    if (releaseSourceLoading.value) return '正在读取 Harbor 镜像版本'
    if (releaseSourceError.value) return releaseSourceError.value
    const source = activeReleaseSource.value
    if (!source) return '当前服务暂未读取到 Harbor 镜像版本'
    if (!source.publishable) return source.message || '当前服务没有可供发版的 Harbor 镜像版本'
    if (activeImageTags.value.length === 0) return '当前镜像仓库未发现可供发版的 tag'
    if (!selectedImageTag.value) return '请选择要发版的镜像 tag'
    return ''
  }
  if (!pipelineBound(activeService.value)) return '服务尚未绑定 Jenkins Pipeline，请先绑定后再发版'
  if (jenkinsPipelines.value.length === 0) return '当前产品没有可用 Jenkins Pipeline，请先刷新 Jenkins 基础资源'
  if (!activePipeline.value) return '当前服务绑定的 Jenkins Pipeline 不在产品可用范围内，请重新绑定'
  if (activePipelineParametersUnknown.value) return '当前 Pipeline 参数读取失败，请刷新 Jenkins 基础资源或检查 Jenkins 权限'
  return ''
})
const registryCandidates = computed(() => {
  const candidates = new Set<string>()
  for (const item of managedServices.value) {
    const host = item.privateRegistryHost?.trim()
    if (host && !item.privateRegistryConfirmed && item.imageSource !== 'EXTERNAL') {
      candidates.add(host)
    }
  }
  return [...candidates].sort()
})
const managedRegistryHost = computed(() => product.value?.privateRegistryHost || managedServices.value.find((item) => item.privateRegistryConfirmed && item.privateRegistryHost)?.privateRegistryHost || '')
const managedRegistryConfirmed = computed(() => {
  if (!managedRegistryHost.value) return false
  if (product.value?.privateRegistryHost) return true
  return managedServices.value.some((item) => item.privateRegistryConfirmed)
})
const registryPanelText = computed(() => {
  if (managedRegistryConfirmed.value) return '已确认，后续发版会按该 registry 识别私有镜像'
  if (registryCandidates.value.length > 0) return '从已纳管服务镜像中发现候选 registry，请确认当前产品使用的私有镜像仓库'
  return '当前已纳管服务暂未发现可确认的私有镜像 registry'
})

async function loadPageData() {
  if (loading.value) return
  loading.value = true
  errorMessage.value = ''
  selectedManagedServices.value = []
  selectedDiscoveredServices.value = []
  managedTableRef.value?.clearSelection()
  discoveredTableRef.value?.clearSelection()
  try {
    const [productItems, projectItems] = await Promise.all([listEnvironments(), listProjects()])
    projects.value = projectItems.filter((item) => item.status !== 'DISABLED')
    const activeProjectIds = new Set(projects.value.map((item) => item.id))
    products.value = productItems.filter((item) => !item.projectId || activeProjectIds.has(item.projectId))
    if (selectedProjectId.value && !activeProjectIds.has(selectedProjectId.value)) {
      selectedProjectId.value = ''
    }
    ensureSelectedProduct()
    if (!productId.value) {
      clearProductData()
      errorMessage.value = '请先创建产品'
      return
    }
    const [managedItems, agentItems] = await Promise.all([
      listEnvironmentServices(productId.value),
      listAgents(),
    ])
    product.value = products.value.find((item) => item.id === productId.value) ?? null
    managedServices.value = managedItems
    if (!adoptDialogVisible.value) {
      discoveredServices.value = []
    }
    agents.value = agentItems
    jenkinsInstances.value = await listJenkinsInstances()
    const releaseSource = await listProductReleaseSources()
    releaseSourceServices.value = releaseSource.services
    jenkinsPipelines.value = resolveProductJenkinsPipelines(releaseSource.jenkinsPipelines ?? [])
    if (!selectedRegistryHost.value || !registryCandidates.value.includes(selectedRegistryHost.value)) {
      selectedRegistryHost.value = registryCandidates.value[0] ?? ''
    }
    if (!product.value) {
      errorMessage.value = '未找到当前产品，请返回产品管理确认产品是否存在'
    }
    scheduleManagedServiceInitialSync()
  } catch (error) {
    managedServices.value = []
    discoveredServices.value = []
    releaseSourceServices.value = []
    jenkinsPipelines.value = []
    jenkinsInstances.value = []
    agents.value = []
    errorMessage.value = error instanceof Error ? error.message : '产品服务加载失败'
  } finally {
    loading.value = false
  }
}

function refreshPageDataWhenVisible() {
  if (document.visibilityState !== 'visible' || !productId.value || loading.value) return
  if (product.value?.networkMode !== 'AGENT') return
  void reloadManagedServices({ silent: true })
}

async function listProductReleaseSources() {
  if (!productId.value) return { services: [], jenkinsJobs: [], jenkinsPipelines: [] }
  return listReleaseSources(productId.value, { includeTags: false })
}

async function loadActiveReleaseSourceTags() {
  const service = activeService.value
  if (!service || !productId.value) return
  releaseSourceLoading.value = true
  releaseSourceError.value = ''
  selectedImageTag.value = ''
  try {
    const releaseSource = await listReleaseSources(productId.value, {
      serviceId: service.id,
      includeTags: true,
    })
    const source = releaseSource.services.find((item) => item.serviceId === service.id)
    if (!source) {
      releaseSourceServices.value = releaseSourceServices.value.filter((item) => item.serviceId !== service.id)
      releaseSourceError.value = '当前服务暂未读取到 Harbor 镜像版本'
      return
    }
    releaseSourceServices.value = [
      ...releaseSourceServices.value.filter((item) => item.serviceId !== service.id),
      source,
    ]
    selectedImageTag.value = source.tags[0]?.tag ?? ''
  } catch (error) {
    releaseSourceError.value = error instanceof Error ? error.message : 'Harbor 镜像版本读取失败'
  } finally {
    releaseSourceLoading.value = false
  }
}

function resolveProductJenkinsPipelines(sourcePipelines: JenkinsPipeline[]) {
  return filterPipelinesForCurrentProduct(sourcePipelines)
}

function filterPipelinesForCurrentProduct(pipelines: JenkinsPipeline[]) {
  const viewKeys = new Set(productJenkinsViewNames.value.flatMap(viewKeyCandidates))
  const candidates = new Map<string, JenkinsPipeline>()
  for (const pipeline of pipelines) {
    const normalized = normalizePipelineCandidate(pipeline, productJenkinsViewNames.value, viewKeys)
    if (!normalized) continue
    const candidateKeys = pipelineViewKeyCandidates(normalized)
    if (viewKeys.size === 0 || candidateKeys.some((key) => viewKeys.has(key))) {
      candidates.set(pipelineCandidateKey(normalized), normalized)
    }
  }
  return [...candidates.values()].sort((a, b) => {
    if ((a.view || '') === (b.view || '')) return a.name.localeCompare(b.name)
    return (a.view || '').localeCompare(b.view || '')
  })
}

function normalizePipelineCandidate(pipeline: JenkinsPipeline, viewNames: string[], viewKeys: Set<string>) {
  const name = pipeline.name?.trim()
  if (!name) return null
  const explicitView = pipeline.view?.trim() || ''
  const explicitViewURL = pipeline.viewUrl?.trim() || ''
  const explicitURL = pipeline.url?.trim() || ''
  const hasViewSignal = Boolean(explicitView || explicitViewURL)
  const matchedView = explicitView || inferViewNameFromValue(explicitViewURL)
  if (hasViewSignal && viewKeys.size > 0 && !matchedView) return null
  if (!hasViewSignal && viewKeys.size > 0) return null
  const inferredView = matchedView || (viewNames.length === 1 ? viewNames[0] : '')
  return {
    ...pipeline,
    name,
    view: inferredView,
    viewUrl: explicitViewURL,
    url: explicitURL,
    parameters: pipeline.parameters,
  }
}

function pipelineCandidateKey(pipeline: JenkinsPipeline) {
  return normalizePipelineUrl(pipeline.url || '') || `${pipeline.view || pipeline.viewUrl || ''}\u0000${pipeline.name}`
}

function findProductPipeline(row: ProductService) {
  const trimmed = pipelineName(row).trim()
  if (!trimmed) return null
  const jobURL = pipelineUrl(row).trim()
  const pipelines = filterPipelinesForCurrentProduct(jenkinsPipelines.value)
  if (jobURL) {
    const matchedByURL = pipelines.find((item) => samePipelineUrl(item.url || '', jobURL))
    if (matchedByURL) return matchedByURL
  }
  return pipelines.find((item) => item.name === trimmed) ?? null
}

function samePipelineUrl(left = '', right = '') {
  const normalizedLeft = normalizePipelineUrl(left)
  const normalizedRight = normalizePipelineUrl(right)
  return Boolean(normalizedLeft && normalizedRight && normalizedLeft === normalizedRight)
}

function normalizePipelineUrl(value = '') {
  const trimmed = value.trim().replace(/\/+$/g, '')
  if (!trimmed) return ''
  try {
    return decodeURIComponent(trimmed).replace(/\/+$/g, '')
  } catch {
    return trimmed
  }
}

function normalizeViewKey(value = '') {
  return viewKeyCandidates(value)[0] || ''
}

function pipelineViewKeyCandidates(pipeline: JenkinsPipeline) {
  return uniqueSorted([
    ...viewKeyCandidates(pipeline.view || ''),
    ...viewKeyCandidates(pipeline.viewUrl || ''),
  ])
}

function viewKeyCandidates(value = '') {
  const normalized = value.trim().toLowerCase().replace(/^\/+|\/+$/g, '')
  if (!normalized) return []
  const keys = [normalized]
  let pathValue = normalized
  try {
    const parsed = new URL(normalized)
    if (parsed.pathname) {
      pathValue = parsed.pathname.replace(/^\/+|\/+$/g, '').toLowerCase()
      keys.push(pathValue)
      keys.push(...pathSuffixCandidates(parsed.pathname))
    }
  } catch {
    // value may already be a Jenkins view name or path.
  }
  keys.push(...extractJenkinsViewKeys(pathValue))
  try {
    const decoded = decodeURIComponent(normalized).trim().toLowerCase().replace(/^\/+|\/+$/g, '')
    keys.push(decoded)
    keys.push(...extractJenkinsViewKeys(decoded))
    keys.push(...pathSuffixCandidates(decoded))
  } catch {
    // Keep original normalized key when decoding is not possible.
  }
  return uniqueSorted(keys)
}

function extractJenkinsViewKeys(value = '') {
  const keys: string[] = []
  const pathParts = value.split('/').map((part) => safeDecode(part)).filter(Boolean)
  const viewParts: string[] = []
  for (let index = 0; index < pathParts.length - 1; index += 1) {
    if (pathParts[index] !== 'view') continue
    const viewName = pathParts[index + 1].trim().toLowerCase()
    keys.push(viewName)
    viewParts.push(viewName)
  }
  if (viewParts.length > 1) keys.push(viewParts.join('/'))
  return keys
}

function pathSuffixCandidates(value = '') {
  const parts = value
    .trim()
    .toLowerCase()
    .replace(/^\/+|\/+$/g, '')
    .split('/')
    .map((part) => safeDecode(part))
    .filter(Boolean)
  const keys: string[] = []
  if (parts.length > 0) keys.push(parts[parts.length - 1])
  const viewIndex = parts.lastIndexOf('view')
  if (viewIndex >= 0 && viewIndex + 1 < parts.length) keys.push(parts[viewIndex + 1])
  return keys
}

function safeDecode(value = '') {
  try {
    return decodeURIComponent(value).trim().toLowerCase()
  } catch {
    return value.trim().toLowerCase()
  }
}

function inferViewNameFromValue(value = '') {
  const keys = viewKeyCandidates(value)
  const viewNames = productJenkinsViewNames.value
  return viewNames.find((view) => keys.includes(normalizeViewKey(view))) || ''
}

function ensureSelectedProduct() {
  const queryProductId = typeof route.query.productId === 'string' ? route.query.productId : ''
  const routeProductId = String(route.params.id || '')
  const preferredId = standaloneMode.value ? queryProductId || selectedProductId.value : routeProductId
  const preferredProduct = products.value.find((item) => item.id === preferredId)
  const fallbackProduct = filteredProductsForProject.value[0] ?? products.value[0] ?? null
  const nextProduct = preferredProduct ?? fallbackProduct
  selectedProductId.value = nextProduct?.id || ''
  selectedProjectId.value = nextProduct?.projectId || selectedProjectId.value
}

function clearProductData() {
  product.value = null
  managedServices.value = []
  discoveredServices.value = []
  releaseSourceServices.value = []
  jenkinsPipelines.value = []
  agents.value = []
}

function handleProductChange() {
  const current = products.value.find((item) => item.id === selectedProductId.value)
  if (current?.projectId) {
    selectedProjectId.value = current.projectId
  }
  if (standaloneMode.value) {
    void router.replace({ path: '/services', query: selectedProductId.value ? { productId: selectedProductId.value } : {} })
  }
  void loadPageData()
}

function handleProjectChange() {
  const candidates = filteredProductsForProject.value
  if (!candidates.some((item) => item.id === selectedProductId.value)) {
    selectedProductId.value = candidates[0]?.id || ''
  }
  handleProductChange()
}

function handleDiscoveredSelectionChange(rows: DiscoveredProductService[]) {
  selectedDiscoveredServices.value = rows.filter((item) => !item.managed)
}

function handleManagedSelectionChange(rows: ProductService[]) {
  selectedManagedServices.value = rows
}

function selectableDiscoveredService(row: DiscoveredProductService) {
  return !row.managed
}

async function openAdoptDialog() {
  if (!productId.value) {
    ElMessage.warning('请先选择产品')
    return
  }
  selectedDiscoveredServices.value = []
  adoptDialogVisible.value = true
  try {
    discoveredServices.value = await listDiscoveredEnvironmentServices(productId.value)
  } catch (error) {
    discoveredServices.value = []
    ElMessage.error(error instanceof Error ? error.message : '服务发现失败')
  }
  await nextTick()
  discoveredTableRef.value?.clearSelection()
}

async function refreshManagedServices() {
  if (!productId.value) {
    ElMessage.warning('请先选择产品')
    return
  }
  await syncManagedServicesFromRuntime()
  await nextTick()
  managedTableRef.value?.clearSelection()
}

async function reloadManagedServices(options: { silent?: boolean } = {}) {
  if (!productId.value || syncingManagedServices.value) return
  syncingManagedServices.value = true
  try {
    managedServices.value = await listEnvironmentServices(productId.value)
  } catch (error) {
    if (!options.silent) {
      ElMessage.error(error instanceof Error ? error.message : '服务列表刷新失败')
    }
  } finally {
    syncingManagedServices.value = false
  }
}

async function syncManagedServicesFromRuntime(options: { silent?: boolean } = {}) {
  if (!productId.value || syncingManagedServices.value) return
  syncingManagedServices.value = true
  try {
    managedServices.value = await syncEnvironmentServices(productId.value)
    if (!options.silent) {
      ElMessage.success('已刷新实际运行镜像状态')
    }
  } catch (error) {
    if (!options.silent) {
      ElMessage.error(error instanceof Error ? error.message : '服务运行状态刷新失败')
    }
  } finally {
    syncingManagedServices.value = false
  }
}

function startManagedServiceSyncTimer() {
  stopManagedServiceSyncTimer()
  managedServiceSyncTimer = window.setInterval(() => {
    if (document.visibilityState !== 'visible' || !productId.value || loading.value) return
    if (product.value?.networkMode !== 'AGENT') return
    void reloadManagedServices({ silent: true })
  }, MANAGED_SERVICE_SYNC_INTERVAL_MS)
}

function stopManagedServiceSyncTimer() {
  if (!managedServiceSyncTimer) return
  window.clearInterval(managedServiceSyncTimer)
  managedServiceSyncTimer = undefined
}

function scheduleManagedServiceInitialSync() {
  stopManagedServiceInitialSyncTimer()
  if (!productId.value) return
  managedServiceInitialSyncTimer = window.setTimeout(() => {
    if (document.visibilityState !== 'visible' || !productId.value || loading.value) return
    void syncManagedServicesFromRuntime({ silent: true })
  }, 500)
}

function stopManagedServiceInitialSyncTimer() {
  if (!managedServiceInitialSyncTimer) return
  window.clearTimeout(managedServiceInitialSyncTimer)
  managedServiceInitialSyncTimer = undefined
}

async function adoptSelectedServices() {
  if (selectedDiscoveredServices.value.length === 0) return
  adopting.value = true
  try {
    managedServices.value = await adoptEnvironmentServices(productId.value, selectedDiscoveredServices.value)
    ElMessage.success('服务已纳管，可继续选择其他发现服务')
    await loadPageData()
    adoptDialogVisible.value = false
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '服务纳管失败')
  } finally {
    adopting.value = false
  }
}

async function confirmManagedRegistry() {
  const candidate = selectedRegistryHost.value
  if (!candidate || managedRegistryConfirmed.value) return
  try {
    await ElMessageBox.confirm(
      `当前已纳管服务发现私有镜像 registry：${candidate}。确认后平台会把它作为该产品的私有镜像仓库，用于后续发版识别。`,
      '确认产品私有 registry',
      {
        confirmButtonText: '确认',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )
  } catch {
    return
  }
  confirmingRegistry.value = true
  try {
    managedServices.value = await confirmEnvironmentServiceRegistry(productId.value, candidate)
    if (product.value) {
      product.value.privateRegistryHost = candidate
    }
    ElMessage.success('私有 registry 已确认')
    await loadPageData()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '私有 registry 确认失败')
  } finally {
    confirmingRegistry.value = false
  }
}

async function removeSelectedManagedServices() {
  const serviceIds = selectedManagedServices.value.map((item) => item.id)
  if (serviceIds.length === 0) return
  try {
    await ElMessageBox.confirm(
      `确认将选中的 ${serviceIds.length} 个服务移出当前产品的发版与部署范围？`,
      '移除纳管服务',
      {
        confirmButtonText: '移除',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )
  } catch {
    return
  }
  removing.value = true
  try {
    managedServices.value = await removeEnvironmentServices(productId.value, serviceIds)
    ElMessage.success(`已移除 ${serviceIds.length} 个纳管服务`)
    await loadPageData()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '移除纳管失败')
  } finally {
    removing.value = false
  }
}

function openPipelineDialog(row: ProductService) {
	activeService.value = row
	const currentPipeline = findProductPipeline(row)
	pipelineForm.value = {
		pipelineKey: currentPipeline ? pipelineCandidateKey(currentPipeline) : '',
	}
  pipelineDialogVisible.value = true
}

async function openServiceDetail(row: ProductService) {
  activeService.value = row
  serviceDetailVisible.value = true
  await loadServiceReleaseHistory()
}

async function loadServiceReleaseHistory() {
  if (!productId.value || !activeService.value) {
    serviceReleaseHistory.value = []
    return
  }
  loadingServiceReleases.value = true
  try {
    serviceReleaseHistory.value = await listServiceReleases(productId.value, activeService.value.id)
  } catch (error) {
    serviceReleaseHistory.value = []
    ElMessage.error(error instanceof Error ? error.message : '服务发版历史加载失败')
  } finally {
    loadingServiceReleases.value = false
  }
}

async function submitPipelineBinding() {
  if (!activeService.value) return
  const pipeline = availablePipelinesForActiveService.value.find((item) => pipelineCandidateKey(item) === pipelineForm.value.pipelineKey)
  if (!pipeline) {
    ElMessage.warning('请选择 Jenkins Pipeline')
    return
  }
  const jobName = pipeline.name.trim()
  bindingPipeline.value = true
  try {
    const updated = await bindEnvironmentServicePipeline(productId.value, activeService.value.id, {
      jenkinsJobName: jobName,
      jenkinsJobUrl: pipeline.url?.trim() || '',
    })
    managedServices.value = managedServices.value.map((item) => (item.id === updated.id ? updated : item))
    activeService.value = updated
    pipelineDialogVisible.value = false
    ElMessage.success('Pipeline 已绑定')
    await loadPageData()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Pipeline 绑定失败')
  } finally {
    bindingPipeline.value = false
  }
}

function openReleaseDialog(row: ProductService) {
  activeService.value = row
  releaseMode.value = pipelineBound(row) ? 'JENKINS_JOB' : 'LOCAL_HARBOR_IMAGE'
  releaseSourceError.value = ''
  releaseParameters.value = {}
  for (const param of activePipelineParameters.value) {
    releaseParameters.value[param.name] = param.defaultValue ?? ''
  }
  selectedImageTag.value = activeImageTags.value[0]?.tag ?? ''
  releaseDialogVisible.value = true
  if (releaseMode.value === 'LOCAL_HARBOR_IMAGE') {
    void loadActiveReleaseSourceTags()
  }
}

async function submitRelease() {
  if (!activeService.value || releaseWarning.value) return
  if (releaseMode.value === 'LOCAL_HARBOR_IMAGE') {
    await submitImageRelease()
    return
  }
  const parameters = buildReleaseParameters()
  for (const param of activePipelineParameters.value) {
    const value = parameters[param.name]
    if (param.required && (!value || !value.trim())) {
      ElMessage.warning(`请填写 Jenkins 参数：${param.name}`)
      return
    }
  }
  creatingRelease.value = true
  try {
    const result = await createRelease({
      type: 'SERVICE_RELEASE',
      releaseSource: 'JENKINS_JOB',
      targetEnvironmentId: productId.value,
      agentId: product.value?.networkMode === 'AGENT' ? activeAgent.value?.id || '' : '',
      serviceIds: [activeService.value.id],
      jenkins: {
        jobName: pipelineName(activeService.value),
        jobUrl: activePipeline.value?.url?.trim() || pipelineUrl(activeService.value),
        branch: releaseBranchValue(parameters),
        parameters,
      },
      options: {},
    })
    releaseDialogVisible.value = false
    creatingRelease.value = false
    await router.push({ path: `/releases/${result.id}` })
    ElMessage.success('已进入发版日志')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '发版单创建失败')
  } finally {
    creatingRelease.value = false
  }
}

async function submitImageRelease() {
  if (!activeService.value || !activeReleaseSource.value || !selectedImageTag.value) return
  const selectedTag = activeImageTags.value.find((item) => item.tag === selectedImageTag.value)
  creatingRelease.value = true
  try {
    const result = await createRelease({
      type: 'SERVICE_RELEASE',
      releaseSource: 'LOCAL_HARBOR_IMAGE',
      targetEnvironmentId: productId.value,
      agentId: product.value?.networkMode === 'AGENT' ? activeAgent.value?.id || '' : '',
      serviceIds: [activeService.value.id],
      image: {
        repository: activeReleaseSource.value.imageRepository,
        tag: selectedImageTag.value,
        digest: selectedTag?.digest || '',
      },
      options: {},
    })
    releaseDialogVisible.value = false
    creatingRelease.value = false
    await router.push({ path: `/releases/${result.id}` })
    ElMessage.success('已进入发版日志')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '发版失败')
  } finally {
    creatingRelease.value = false
  }
}

function buildReleaseParameters() {
  return { ...releaseParameters.value }
}

function isBranchParameterName(name: string) {
  return [
    'branch',
    'branch_name',
    'branchname',
    'git_branch',
    'gitbranch',
    'git_branch_name',
    'gitbranchname',
    'ref',
    'git_ref',
  ].includes(name.trim().toLowerCase())
}

function releaseBranchValue(parameters: Record<string, string>) {
  const branchParam = activeBranchParameter.value
  if (branchParam) return parameters[branchParam.name]?.trim() || ''
  return activeService.value?.jenkinsBranch?.trim() || ''
}

function containerTypeLabel(type = '') {
  return type === 'INIT' ? '初始化容器' : '普通容器'
}

function containerTagType(type = ''): '' | 'success' | 'info' | 'warning' | 'danger' | 'primary' {
  return type === 'INIT' ? 'warning' : 'primary'
}

function imageSourceLabel(row: ProductService | DiscoveredProductService, scope: 'managed' | 'discovered') {
  const source = row.imageSource || ''
  if (row.privateRegistryHost && !row.privateRegistryConfirmed && source !== 'EXTERNAL') {
    return scope === 'managed' ? '待确认私有镜像' : '候选私有镜像'
  }
  if (source === 'PRIVATE') return '私有镜像'
  if (source === 'UNMATCHED_PRIVATE') return '私有项目未纳管'
  return '公共/外部镜像'
}

function imageSourceTip(row: ProductService | DiscoveredProductService, scope: 'managed' | 'discovered') {
  const source = row.imageSource || ''
  if (row.privateRegistryHost && !row.privateRegistryConfirmed && source !== 'EXTERNAL') {
    if (scope === 'managed') {
      return `候选私有 registry：${row.privateRegistryHost}，确认后用于后续发版识别`
    }
    return `候选私有 registry：${row.privateRegistryHost}，服务纳管后可在已纳管服务中确认`
  }
  if (source === 'PRIVATE') return '镜像 registry 与产品 Harbor 匹配，且 project 已纳管'
  if (source === 'UNMATCHED_PRIVATE') return '镜像 registry 与产品 Harbor 匹配，但 project 未纳管到当前产品'
  return '镜像 registry 不属于当前产品 Harbor'
}

function imageSourceTagType(source = ''): '' | 'success' | 'info' | 'warning' | 'danger' | 'primary' {
  if (source === 'PRIVATE') return 'success'
  if (source === 'UNMATCHED_PRIVATE') return 'warning'
  return 'info'
}

function releaseSourceOf(row: ProductService) {
  return releaseSourceByServiceId.value.get(row.id)
}

function pipelineName(row: ProductService) {
  return row.jenkinsJobName || releaseSourceOf(row)?.jenkinsJobName || ''
}

function pipelineUrl(row: ProductService) {
  return row.jenkinsJobUrl || releaseSourceOf(row)?.jenkinsJobUrl || ''
}

function pipelineHasStoredBinding(row: ProductService) {
  return Boolean(row.jenkinsPipelineBound || pipelineName(row))
}

function pipelineBound(row: ProductService) {
  return pipelineHasStoredBinding(row) && Boolean(findProductPipeline(row))
}

function pipelineBindingInvalid(row: ProductService) {
  return pipelineHasStoredBinding(row) && !pipelineBound(row)
}

function pipelineBindingLabel(row: ProductService) {
  if (pipelineBound(row)) return '已绑定'
  if (pipelineBindingInvalid(row)) return '绑定失效'
  return '未绑定'
}

function pipelineBindingTagType(row: ProductService): '' | 'success' | 'info' | 'warning' | 'danger' | 'primary' {
  if (pipelineBound(row)) return 'success'
  if (pipelineBindingInvalid(row)) return 'danger'
  return 'warning'
}

function pipelineBindingTip(row: ProductService) {
  if (pipelineBound(row)) return '当前 Pipeline 属于产品绑定的 Jenkins view，可以发版'
  if (pipelineBindingInvalid(row)) return '服务保存了 Pipeline 绑定，但当前产品 Jenkins view 未发现该 Pipeline，请刷新 Jenkins 基础资源或重新绑定'
  return '服务尚未绑定 Jenkins Pipeline'
}

function pipelineOptionLabel(pipeline: JenkinsPipeline) {
  const prefix = pipeline.view ? `${pipeline.view} / ` : ''
  const parameters = pipeline.parameters
  const parameterText = Array.isArray(parameters) ? ` / ${parameters.length} 个参数` : ' / 参数待探测'
  return `${prefix}${pipeline.name}${parameterText}`
}

function pipelineParameterPlaceholder(param: { type?: string; description?: string; defaultValue?: string }) {
  if (param.defaultValue) return `默认值：${param.defaultValue}`
  return param.description || param.type || '请输入参数值'
}

function pipelineParameterTip(param: { required?: boolean; description?: string; type?: string }) {
  const parts = []
  if (param.required) parts.push('必填')
  if (param.type) parts.push(param.type)
  if (param.description) parts.push(param.description)
  return parts.join(' / ')
}

function imageTagOptionLabel(tag: ReleaseImageTag) {
  const parts = [tag.tag]
  if (tag.updatedAt) parts.push(formatTime(tag.updatedAt))
  if (tag.digest) parts.push(tag.digest.slice(0, 18))
  return parts.join(' / ')
}

function releaseImageVersion(row: ReleaseOrder) {
  if (row.imageRepository && row.imageTag) return `${row.imageRepository}:${row.imageTag}`
  if (row.imageRepository || row.imageTag) return row.imageRepository || row.imageTag || '-'
  return '待环境确认'
}

function releaseImageTip(row: ReleaseOrder) {
  if (row.imageDigest) return row.imageDigest
  if (row.imageRepository && row.imageTag) return row.releaseSource === 'LOCAL_HARBOR_IMAGE' ? '选择的 Harbor 镜像' : '发版目标镜像'
  return 'Jenkins 发版后以实际 K8s/Agent 上报为准'
}

function releaseStatusLabel(status = '') {
  const normalized = status.trim().toUpperCase()
  const labels: Record<string, string> = {
    CREATED: '已创建',
    RUNNING: '执行中',
    SUCCESS: '成功',
    FAILED: '失败',
    PENDING_IMAGE_SYNC: '等待镜像同步',
    JENKINS_TRIGGERING: 'Jenkins 触发中',
    JENKINS_QUEUED: 'Jenkins 排队',
    QUEUED: '排队',
    BUILDING: '构建中',
    ABORTED: '已中止',
  }
  return labels[normalized] || status || '-'
}

function releaseStatusTagType(status = ''): '' | 'success' | 'info' | 'warning' | 'danger' | 'primary' {
  const normalized = status.trim().toUpperCase()
  if (['SUCCESS', 'COMPLETED'].includes(normalized)) return 'success'
  if (['FAILED', 'ABORTED'].includes(normalized)) return 'danger'
  if (['RUNNING', 'BUILDING', 'JENKINS_TRIGGERING', 'JENKINS_QUEUED', 'QUEUED', 'PENDING_IMAGE_SYNC'].includes(normalized)) return 'warning'
  return 'info'
}

function formatTime(value = '') {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (num: number) => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

function uniqueSorted(values: string[]) {
  return [...new Set(values.map((item) => item.trim()).filter(Boolean))].sort()
}

function scopedValues(row: EnvironmentInfo | null, resourceType: 'K8S' | 'HARBOR' | 'JENKINS') {
  if (!row?.bindings) return []
  return uniqueSorted(
    row.bindings
      .filter((item) => item.resourceType === resourceType && item.bindingRole !== 'RUNTIME_TARGET')
      .map((item) => item.scopeValue),
  )
}

watch([releaseMode, activeReleaseSource], () => {
  releaseSourceError.value = ''
  if (releaseMode.value !== 'LOCAL_HARBOR_IMAGE') return
  if (activeService.value && activeImageTags.value.length === 0 && !releaseSourceLoading.value) {
    void loadActiveReleaseSourceTags()
    return
  }
  if (!activeImageTags.value.some((tag) => tag.tag === selectedImageTag.value)) {
    selectedImageTag.value = activeImageTags.value[0]?.tag ?? ''
  }
})

onMounted(() => {
  void loadPageData()
  startManagedServiceSyncTimer()
  window.addEventListener('focus', refreshPageDataWhenVisible)
  document.addEventListener('visibilitychange', refreshPageDataWhenVisible)
})

onUnmounted(() => {
  stopManagedServiceSyncTimer()
  stopManagedServiceInitialSyncTimer()
  window.removeEventListener('focus', refreshPageDataWhenVisible)
  document.removeEventListener('visibilitychange', refreshPageDataWhenVisible)
})
</script>

<style scoped>
.head-actions,
.service-actions {
  align-items: center;
  display: flex;
  gap: 10px;
}

.page-head {
  align-items: center;
  display: flex;
  gap: 12px;
  margin-bottom: 8px;
  padding: 0 2px;
}

.page-head h1 {
  color: #172033;
  font-size: 20px;
  margin-bottom: 4px;
}

.page-head p {
  color: #667085;
  font-size: 13px;
  margin: 0;
}

.service-alert {
  margin-bottom: 4px;
}

.service-control-bar {
  align-items: center;
  background: #fff;
  border: 1px solid #e4e8f0;
  border-radius: 6px;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
  padding: 8px;
}

.product-switcher {
  align-items: center;
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(2, minmax(210px, 300px));
  min-width: 0;
}

.product-switcher label {
  align-items: center;
  display: flex;
  gap: 8px;
  margin: 0;
}

.product-switcher span {
  color: #606a7b;
  font-size: 13px;
}

.product-switcher .el-select {
  width: 100%;
}

.service-summary {
  align-items: center;
  color: #606a7b;
  display: grid;
  flex: 1 1 auto;
  gap: 8px;
  grid-template-columns: repeat(2, minmax(76px, auto)) minmax(220px, 1fr);
  justify-content: flex-end;
  min-width: 280px;
}

.service-summary > span {
  background: transparent;
  border: 0;
  font-size: 12px;
  line-height: 16px;
  padding: 0;
  white-space: nowrap;
}

.service-summary strong {
  color: #2f3847;
  font-size: 13px;
  line-height: 16px;
}

.pipeline-summary {
  align-items: center;
  display: flex;
  gap: 8px;
  max-width: min(420px, 100%);
}

.pipeline-summary strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-list-panel {
  background: #fff;
  border: 1px solid #e4e8f0;
  border-radius: 6px;
  padding: 6px 6px 8px;
}

.panel-head {
  align-items: center;
  background: #fafbfc;
  border: 1px solid #edf1f6;
  border-radius: 4px;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin: 0 0 8px;
  padding: 6px 8px;
}

.panel-title {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.panel-head strong {
  color: #172033;
  font-size: 15px;
}

.panel-head span,
.container-name,
.image-meta span,
.version-source-cell span,
.pipeline-cell span,
.service-actions span {
  color: #7a8294;
  font-size: 12px;
}

.service-actions {
  align-items: center;
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  gap: 6px;
  justify-content: flex-end;
  min-width: max-content;
}

.service-actions .el-button {
  margin-left: 0;
}

.service-table {
  --el-table-header-bg-color: #f8fafc;
  border: 1px solid #edf1f6;
  border-radius: 6px;
  overflow: hidden;
}

.service-table :deep(.cell) {
  overflow-wrap: anywhere;
  padding-left: 8px;
  padding-right: 8px;
}

.service-table :deep(.el-table__body td) {
  padding: 7px 0;
}

.service-table :deep(.el-table__header th) {
  color: #606a7b;
  font-weight: 600;
}

.service-table :deep(.el-table__row) {
  --el-table-row-hover-bg-color: #f7f9fc;
}

.registry-panel {
  align-items: center;
  background: #fbfcfe;
  border: 1px solid #edf1f6;
  border-radius: 6px;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  padding: 10px 12px;
}

.registry-panel > div:first-child {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.registry-panel strong {
  color: #2f3847;
  font-size: 13px;
}

.registry-panel span {
  color: #606a7b;
  font-size: 12px;
  line-height: 18px;
}

.registry-confirm,
.registry-confirmed {
  align-items: center;
  display: flex;
  flex: 0 0 auto;
  gap: 8px;
}

.registry-select {
  width: min(360px, 48vw);
}

.service-filter {
  display: grid;
  gap: 6px;
  grid-template-columns: minmax(240px, 1.8fr) repeat(5, minmax(92px, 1fr));
  margin-bottom: 8px;
}

.service-name-cell,
.image-cell,
.image-meta,
.version-source-cell,
.pipeline-cell {
  display: flex;
  flex-direction: row;
  align-items: center;
  flex-wrap: wrap;
  gap: 5px 8px;
}

.service-name-cell strong {
  color: #2f3847;
  font-size: 14px;
}

.service-link {
  font-size: 14px;
  font-weight: 600;
  height: auto;
  line-height: 18px;
  padding: 0;
}

.service-name-cell span,
.image-cell span,
.version-source-cell span {
  color: #606a7b;
  font-size: 12px;
  line-height: 18px;
}

.image-cell > span,
.pipeline-cell > span,
.version-source-cell > span {
  display: block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.image-meta {
  align-items: flex-start;
}

.pipeline-cell .el-button {
  padding: 0;
}

.dialog-control {
  width: 100%;
}

.dialog-context {
  background: #f8fafc;
  border: 1px solid #e4e8f0;
  border-radius: 6px;
  display: grid;
  gap: 10px;
  grid-template-columns: minmax(0, 1fr) 120px;
  margin-bottom: 14px;
  padding: 10px 12px;
}

.dialog-context div {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.dialog-context span {
  color: #7a8294;
  font-size: 12px;
}

.dialog-context strong {
  color: #2f3847;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.form-tip {
  color: #7a8294;
  font-size: 12px;
  line-height: 18px;
  margin-top: 6px;
}

.dialog-alert {
  margin-top: 12px;
}

.inline-alert {
  margin-top: 8px;
}

.dialog-table-head {
  align-items: center;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.dialog-table-head span {
  color: #606a7b;
  font-size: 13px;
}

.dialog-table-head strong {
  color: #2f3847;
  flex: 0 0 auto;
  font-size: 13px;
}

.release-summary {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: 16px;
}

.release-summary div {
  background: #f7f9fc;
  border: 1px solid #e4e8f0;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  padding: 10px 12px;
}

.release-summary span {
  color: #7a8294;
  font-size: 12px;
}

.release-summary strong {
  color: #2f3847;
  font-size: 13px;
  overflow-wrap: anywhere;
}

.release-parameters {
  margin-top: 14px;
}

.section-title {
  color: #2f3847;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 10px;
}

.detail-head {
  align-items: center;
  border-bottom: 1px solid #edf1f6;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
  padding-bottom: 12px;
}

.detail-head > div:first-child {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.detail-head strong {
  color: #172033;
  font-size: 18px;
}

.detail-head span {
  color: #667085;
  font-size: 13px;
}

.detail-actions {
  align-items: center;
  display: flex;
  flex: 0 0 auto;
  gap: 8px;
}

.service-detail-grid {
  display: grid;
  gap: 8px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: 18px;
}

.service-detail-grid div {
  background: #f8fafc;
  border: 1px solid #edf1f6;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  padding: 9px 10px;
}

.service-detail-grid .span-2 {
  grid-column: span 2;
}

.service-detail-grid span {
  color: #7a8294;
  font-size: 12px;
}

.service-detail-grid strong {
  color: #2f3847;
  font-size: 13px;
  overflow-wrap: anywhere;
}

.history-head {
  align-items: center;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.history-head strong {
  color: #172033;
  font-size: 15px;
}

.release-history-table {
  border: 1px solid #edf1f6;
  border-radius: 6px;
  overflow: hidden;
}

.history-jenkins-cell {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.history-jenkins-cell a,
.release-history-table a {
  color: #2563eb;
  text-decoration: none;
}

@media (max-width: 900px) {
  .service-filter,
  .release-summary,
  .dialog-table-head {
    grid-template-columns: 1fr;
    align-items: stretch;
    flex-direction: column;
  }

  .service-control-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .product-switcher {
    grid-template-columns: 1fr;
  }

  .service-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    justify-content: stretch;
    width: 100%;
  }

  .panel-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .registry-panel {
    align-items: stretch;
    flex-direction: column;
  }

  .registry-confirm {
    align-items: stretch;
    flex-direction: column;
  }

  .registry-select {
    width: 100%;
  }

  .dialog-context {
    grid-template-columns: 1fr;
  }

  .detail-head {
    align-items: stretch;
    flex-direction: column;
  }

  .detail-actions {
    justify-content: flex-start;
  }

  .service-detail-grid,
  .service-detail-grid .span-2 {
    grid-template-columns: 1fr;
    grid-column: auto;
  }
}
</style>
