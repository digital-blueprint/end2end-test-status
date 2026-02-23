<template>
  <h1>All Results</h1>

  <div class="filters">
    <div>
      <label>Status&nbsp;</label>
      <select v-model="filterStatus" @change="load">
        <option value="">All</option>
        <option value="success">Success</option>
        <option value="failed">Failed</option>
      </select>
    </div>
    <div>
      <label>Browser&nbsp;</label>
      <select v-model="filterBrowser" @change="load">
        <option value="">All</option>
        <option v-for="b in browsers" :key="b" :value="b">{{ b }}</option>
      </select>
    </div>
    <div>
      <label>Limit&nbsp;</label>
      <select v-model="limit" @change="load">
        <option value="50">50</option>
        <option value="100">100</option>
        <option value="200">200</option>
        <option value="500">500</option>
      </select>
    </div>
  </div>

  <div v-if="loading" class="loading">Loading...</div>
  <div v-else-if="error" class="loading">{{ error }}</div>
  <div v-else-if="results.length === 0" class="loading">No results found.</div>
  <div v-else class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>Status</th>
          <th>Project</th>
          <th>Spec</th>
          <th>Browser</th>
          <th>Pipeline</th>
          <th>Job</th>
          <th>Date</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="r in results" :key="r.id">
          <td><span class="badge" :class="statusClass(r.status)">{{ r.status }}</span></td>
          <td>
            <RouterLink
              :to="`/project/${encodeURIComponent(r.project)}`"
              style="color:#38bdf8"
            >{{ r.project }}</RouterLink>
          </td>
          <td class="wrap" style="max-width:260px;font-family:monospace;font-size:0.8rem">{{ r.spec }}</td>
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

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { fetchJSON, formatDate, statusClass, type TestResult } from '../types'

const results = ref<TestResult[]>([])
const loading = ref(true)
const error = ref('')

const filterStatus = ref('')
const filterBrowser = ref('')
const limit = ref('100')

const browsers = computed(() => [...new Set(results.value.map(r => r.browser).filter(Boolean))])

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams({ limit: limit.value })
    if (filterStatus.value) params.set('status', filterStatus.value)
    if (filterBrowser.value) params.set('browser', filterBrowser.value)
    results.value = await fetchJSON<TestResult[]>(`/api/results?${params}`)
  } catch (e) {
    error.value = 'Failed to load results.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
