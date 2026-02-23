<template>
  <div class="back" @click="router.back()">&#8592; Back</div>

  <h1>{{ projectName }}</h1>

  <div v-if="loading" class="loading">Loading results...</div>
  <div v-else-if="error" class="loading">{{ error }}</div>
  <template v-else>
    <div class="stats">
      <div class="stat-box">
        <div class="label">Total Runs</div>
        <div class="value">{{ results.length }}</div>
      </div>
      <div class="stat-box">
        <div class="label">Passing</div>
        <div class="value" style="color:#4ade80">{{ passingCount }}</div>
      </div>
      <div class="stat-box">
        <div class="label">Failing</div>
        <div class="value" style="color:#f87171">{{ failingCount }}</div>
      </div>
    </div>

    <div class="filters">
      <div>
        <label>Status&nbsp;</label>
        <select v-model="filterStatus">
          <option value="">All</option>
          <option value="success">Success</option>
          <option value="failed">Failed</option>
        </select>
      </div>
      <div>
        <label>Browser&nbsp;</label>
        <select v-model="filterBrowser">
          <option value="">All</option>
          <option v-for="b in browsers" :key="b" :value="b">{{ b }}</option>
        </select>
      </div>
      <div>
        <label>Spec&nbsp;</label>
        <input v-model="filterSpec" placeholder="Filter spec..." style="width:200px" />
      </div>
    </div>

    <div v-if="filtered.length === 0" class="loading">No results match your filters.</div>
    <div v-else class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Status</th>
            <th>Spec</th>
            <th>Browser</th>
            <th>Pipeline</th>
            <th>Job</th>
            <th>Date</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in filtered" :key="r.id">
            <td><span class="badge" :class="statusClass(r.status)">{{ r.status }}</span></td>
            <td class="wrap" style="max-width:300px;font-family:monospace;font-size:0.8rem">{{ r.spec }}</td>
            <td>{{ r.browser || '—' }}</td>
            <td>{{ r.pipeline_id || '—' }}</td>
            <td>
              <a v-if="r.job_url" :href="r.job_url" target="_blank" rel="noopener" class="ext">#{{ r.job_id }}</a>
              <span v-else>{{ r.job_id || '—' }}</span>
            </td>
            <td>{{ formatDate(r.created_at) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </template>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { fetchJSON, formatDate, statusClass, type TestResult } from '../types'

const route = useRoute()
const router = useRouter()

const projectName = computed(() => decodeURIComponent(route.params.name as string))

const results = ref<TestResult[]>([])
const loading = ref(true)
const error = ref('')

const filterStatus = ref('')
const filterBrowser = ref('')
const filterSpec = ref('')

const browsers = computed(() => [...new Set(results.value.map(r => r.browser).filter(Boolean))])

const filtered = computed(() =>
  results.value.filter(r => {
    if (filterStatus.value && r.status !== filterStatus.value) return false
    if (filterBrowser.value && r.browser !== filterBrowser.value) return false
    if (filterSpec.value && !r.spec.toLowerCase().includes(filterSpec.value.toLowerCase())) return false
    return true
  })
)

const passingCount = computed(() => results.value.filter(r => r.status === 'success').length)
const failingCount = computed(() => results.value.filter(r => r.status !== 'success').length)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams({ limit: '500' })
    results.value = await fetchJSON<TestResult[]>(
      `/api/projects/${encodeURIComponent(projectName.value)}/results?${params}`
    )
  } catch (e) {
    error.value = 'Failed to load results.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(() => route.params.name, load)
</script>
