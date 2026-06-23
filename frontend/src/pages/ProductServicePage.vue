<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>产品服务</h1>
        <p>{{ productTitle }}</p>
      </div>
      <div class="head-actions">
        <el-button @click="goBack">返回产品管理</el-button>
        <el-button :loading="loading" @click="loadPageData">刷新</el-button>
      </div>
    </div>

    <el-alert v-if="errorMessage" class="service-alert" type="warning" :closable="false" :title="errorMessage" />

    <div class="metric-grid service-metrics">
      <el-card shadow="never" class="metric-card">
        <div class="metric-label">已纳管服务</div>
        <div class="metric-value">{{ managedServices.length }}</div>
        <div class="metric-foot">纳入后续发版与部署范围</div>
      </el-card>
      <el-card shadow="never" class="metric-card">
        <div class="metric-label">发现服务</div>
        <div class="metric-value">{{ discoveredServices.length }}</div>
        <div class="metric-foot">来自 K8s Deployment / StatefulSet / DaemonSet 容器</div>
      </el-card>
      <el-card shadow="never" class="metric-card">
        <div class="metric-label">可纳管服务</div>
        <div class="metric-value">{{ unmanagedServices.length }}</div>
        <div class="metric-foot">可多次选择并纳管</div>
      </el-card>
      <el-card shadow="never" class="metric-card">
        <div class="metric-label">可发版服务</div>
        <div class="metric-value">{{ publishableServiceCount }}</div>
        <div class="metric-foot">来自已确认私有 Harbor tag</div>
      </el-card>
    </div>

    <el-card shadow="never">
      <template #header>
        <div class="panel-head">
          <div>
            <strong>已纳管服务</strong>
            <span>{{ managedServices.length }} 个容器服务</span>
          </div>
          <div class="service-actions">
            <span>已选择 {{ selectedManagedServices.length }} 个</span>
            <el-button
              type="danger"
              :disabled="selectedManagedServices.length === 0"
              :loading="removing"
              @click="removeSelectedManagedServices"
            >
              移除所选服务
            </el-button>
          </div>
        </div>
      </template>
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
        v-loading="loading"
        :data="managedServices"
        class="service-table"
        @selection-change="handleManagedSelectionChange"
      >
        <el-table-column type="selection" width="48" />
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
        <el-table-column label="版本来源" min-width="220">
          <template #default="{ row }">
            <div class="version-source-cell">
              <el-tooltip :content="versionSourceTip(row)" placement="top">
                <el-tag size="small" :type="versionSourceTagType(row)" effect="light">
                  {{ versionSourceLabel(row) }}
                </el-tag>
              </el-tooltip>
              <span>{{ versionSourceMeta(row) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="副本" width="90">
          <template #default="{ row }">{{ row.readyReplicas }}/{{ row.replicas }}</template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <div class="panel-head">
          <div>
            <strong>发现服务</strong>
            <span>{{ discoveredServices.length }} 个容器服务</span>
          </div>
          <div class="service-actions">
            <span>已选择 {{ selectedDiscoveredServices.length }} 个</span>
            <el-button
              type="primary"
              :disabled="selectedDiscoveredServices.length === 0"
              :loading="adopting"
              @click="adoptSelectedServices"
            >
              纳管所选服务
            </el-button>
          </div>
        </div>
      </template>
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
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type TableInstance } from 'element-plus'
import {
  adoptEnvironmentServices,
  confirmEnvironmentServiceRegistry,
  listDiscoveredEnvironmentServices,
  listEnvironments,
  listEnvironmentServices,
  removeEnvironmentServices,
  type DiscoveredProductService,
  type EnvironmentInfo,
  type ProductService,
} from '@/api/environments'
import { listReleaseSources, type ReleaseSourceService } from '@/api/releases'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const adopting = ref(false)
const confirmingRegistry = ref(false)
const removing = ref(false)
const errorMessage = ref('')
const product = ref<EnvironmentInfo | null>(null)
const managedServices = ref<ProductService[]>([])
const discoveredServices = ref<DiscoveredProductService[]>([])
const releaseSourceServices = ref<ReleaseSourceService[]>([])
const selectedManagedServices = ref<ProductService[]>([])
const selectedDiscoveredServices = ref<DiscoveredProductService[]>([])
const selectedRegistryHost = ref('')
const managedTableRef = ref<TableInstance>()
const discoveredTableRef = ref<TableInstance>()

const productId = computed(() => String(route.params.id || ''))
const productTitle = computed(() => {
  if (!product.value) return '服务只能从探测结果中选择纳管，支持分批多次纳管。'
  const sourceText = product.value.type === 'LOCAL' ? '平台直连探测' : 'Agent 上报探测'
  return `${product.value.name} / ${product.value.code} / ${sourceText}`
})
const unmanagedServices = computed(() => discoveredServices.value.filter((item) => !item.managed))
const releaseSourceByServiceId = computed(() => {
  const services = new Map<string, ReleaseSourceService>()
  for (const service of releaseSourceServices.value) {
    services.set(service.serviceId, service)
  }
  return services
})
const publishableServiceCount = computed(() => releaseSourceServices.value.filter((item) => item.publishable).length)
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
  loading.value = true
  errorMessage.value = ''
  selectedManagedServices.value = []
  selectedDiscoveredServices.value = []
  managedTableRef.value?.clearSelection()
  discoveredTableRef.value?.clearSelection()
  try {
    const [products, managedItems, discoveredItems] = await Promise.all([
      listEnvironments(),
      listEnvironmentServices(productId.value),
      listDiscoveredEnvironmentServices(productId.value),
    ])
    product.value = products.find((item) => item.id === productId.value) ?? null
    managedServices.value = managedItems
    discoveredServices.value = discoveredItems
    releaseSourceServices.value = await listProductReleaseSources()
    if (!selectedRegistryHost.value || !registryCandidates.value.includes(selectedRegistryHost.value)) {
      selectedRegistryHost.value = registryCandidates.value[0] ?? ''
    }
    if (!product.value) {
      errorMessage.value = '未找到当前产品，请返回产品管理确认产品是否存在'
    }
  } catch (error) {
    managedServices.value = []
    discoveredServices.value = []
    releaseSourceServices.value = []
    errorMessage.value = error instanceof Error ? error.message : '产品服务加载失败'
  } finally {
    loading.value = false
  }
}

async function listProductReleaseSources() {
  if (managedServices.value.length === 0) return []
  try {
    const result = await listReleaseSources(productId.value)
    return result.services
  } catch {
    return managedServices.value.map((service) => ({
      serviceId: service.id,
      serviceName: service.name,
      namespace: service.namespace,
      workloadName: service.workloadName,
      workloadType: service.workloadType,
      imageRegistry: service.imageRegistry,
      imageProject: service.imageProject,
      imageRepository: service.imageRepository,
      imageTag: service.imageTag,
      imageSource: service.imageSource,
      privateRegistryHost: service.privateRegistryHost,
      privateRegistryConfirmed: Boolean(service.privateRegistryConfirmed),
      tags: [],
      publishable: false,
      message: '版本来源读取失败',
    }))
  }
}

function goBack() {
  router.push('/environments')
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

async function adoptSelectedServices() {
  if (selectedDiscoveredServices.value.length === 0) return
  adopting.value = true
  try {
    managedServices.value = await adoptEnvironmentServices(productId.value, selectedDiscoveredServices.value)
    ElMessage.success('服务已纳管，可继续选择其他发现服务')
    await loadPageData()
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

function versionSourceLabel(row: ProductService) {
  const source = releaseSourceOf(row)
  if (source?.publishable) return '可发版'
  if (source?.message) return '不可发版'
  return '待确认'
}

function versionSourceTip(row: ProductService) {
  const source = releaseSourceOf(row)
  if (!source) return '版本来源尚未读取'
  if (source.publishable) return `Harbor 已发现 ${source.tags.length} 个镜像 tag`
  return source.message || '请先确认私有镜像 registry 与 Harbor project'
}

function versionSourceMeta(row: ProductService) {
  const source = releaseSourceOf(row)
  if (!source) return '未读取版本来源'
  if (source.publishable) {
    return `当前 ${source.imageTag || '无 tag'} / 可选 ${source.tags.length} 个 tag`
  }
  return source.message || '版本来源未就绪'
}

function versionSourceTagType(row: ProductService): '' | 'success' | 'info' | 'warning' | 'danger' | 'primary' {
  const source = releaseSourceOf(row)
  if (source?.publishable) return 'success'
  if (source?.message) return 'warning'
  return 'info'
}

onMounted(loadPageData)
</script>

<style scoped>
.head-actions,
.service-actions {
  align-items: center;
  display: flex;
  gap: 10px;
}

.service-alert {
  margin-bottom: 4px;
}

.service-metrics {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.panel-head > div {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.panel-head strong {
  color: #2f3847;
  font-size: 15px;
}

.panel-head span,
.container-name,
.image-meta span,
.version-source-cell span,
.service-actions span {
  color: #7a8294;
  font-size: 12px;
}

.service-actions {
  flex-direction: row;
}

.service-table :deep(.cell) {
  overflow-wrap: anywhere;
}

.registry-panel {
  align-items: center;
  background: #f7f9fc;
  border: 1px solid #e4e8f0;
  border-radius: 6px;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  padding: 12px;
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

.service-name-cell,
.image-cell,
.image-meta,
.version-source-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.service-name-cell strong {
  color: #2f3847;
  font-size: 13px;
}

.service-name-cell span,
.image-cell span,
.version-source-cell span {
  color: #606a7b;
  font-size: 12px;
  line-height: 18px;
}

.image-meta {
  align-items: flex-start;
}

@media (max-width: 900px) {
  .service-metrics {
    grid-template-columns: 1fr;
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
}
</style>
