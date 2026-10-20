<template>
  <div v-if="project" class="filters">
    <label for="deployment-environment">Environment</label>
    <select id="deployment-environment" v-model="environment">
      <option value="">All environments</option>
      <option v-for="env in history.environments" :key="env" :value="env">{{ env }}</option>
    </select>
  </div>

  <div v-if="loading" class="loading">Loading deployments...</div>
  <div v-else-if="error" class="loading" role="alert">{{ error }}</div>
  <div v-else-if="history.total === 0" class="loading">No deployments{{ environment ? ' for this environment' : '' }} yet.</div>
  <template v-else>
    <div class="table-wrap">
      <table>
        <thead><tr><th v-if="!project">Project</th><th>Environment</th><th>Version</th><th>Deployed at</th></tr></thead>
        <tbody>
          <tr v-for="deployment in history.deployments" :key="deployment.id">
            <td v-if="!project" class="wrap project-name">
              <RouterLink :to="`/deployments/project/${encodeURIComponent(deployment.project)}`" class="ext">{{ deployment.project }}</RouterLink>
            </td>
            <td class="wrap environment-name">{{ deployment.environment }}</td>
            <td class="wrap deployment-version-cell">
              <VersionLink :deployment="deployment" />
            </td>
            <td><RelativeDate :value="deployment.deployed_at" /></td>
          </tr>
        </tbody>
      </table>
    </div>
    <div class="pagination">
      <button type="button" :disabled="offset === 0" @click="offset -= pageSize">Previous</button>
      <span>{{ offset + 1 }}–{{ offset + history.deployments.length }} of {{ history.total }}</span>
      <button type="button" :disabled="offset + pageSize >= history.total" @click="offset += pageSize">Next</button>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { fetchJSON, type DeploymentHistory as DeploymentHistoryResponse } from '../types'
import RelativeDate from './RelativeDate.vue'
import VersionLink from './VersionLink.vue'

const props = defineProps<{ project?: string }>()
const route = useRoute()
const router = useRouter()
const projectName = computed(() => props.project ?? '')
const history = ref<DeploymentHistoryResponse>({ deployments: [], total: 0, environments: [] })
const environment = computed({
  get: () => props.project && typeof route.query.environment === 'string' ? route.query.environment : '',
  set: value => { void router.replace({ query: { ...route.query, environment: value || undefined } }) },
})
const offset = ref(0)
const pageSize = 50
const loading = ref(true)
const error = ref('')
let requestID = 0

watch(projectName, () => {
  offset.value = 0
  history.value = { deployments: [], total: 0, environments: [] }
})
watch(environment, () => { offset.value = 0 })
watch([projectName, environment, offset], async () => {
  const id = ++requestID
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams({ limit: String(pageSize), offset: String(offset.value) })
    if (projectName.value) params.set('project', projectName.value)
    if (environment.value) params.set('environment', environment.value)
    const result = await fetchJSON<DeploymentHistoryResponse>(`/api/deployments?${params}`)
    if (id === requestID) history.value = result
  } catch {
    if (id === requestID) error.value = 'Failed to load deployment history.'
  } finally {
    if (id === requestID) loading.value = false
  }
}, { immediate: true })
</script>
