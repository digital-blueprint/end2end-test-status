<template>
  <h1>Dashboard</h1>

  <div v-if="loading" class="loading">Loading projects...</div>
  <div v-else-if="error" class="loading">{{ error }}</div>
  <div v-else-if="projects.length === 0" class="loading">
    No projects yet. Send a webhook to get started.
  </div>
  <template v-else>
    <div class="stats">
      <div class="stat-box">
        <div class="label">Projects</div>
        <div class="value">{{ projects.length }}</div>
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

    <div class="grid">
      <RouterLink
        v-for="p in projects"
        :key="p.project"
        :to="`/project/${encodeURIComponent(p.project)}`"
        class="card"
      >
        <div style="display:flex;justify-content:space-between;align-items:flex-start;gap:0.5rem">
          <div style="font-weight:600;font-size:0.95rem;color:var(--heading);word-break:break-word">{{ p.project }}</div>
          <span class="badge" :class="statusClass(p.latest_status)">{{ p.latest_status }}</span>
        </div>
        <div style="margin-top:0.75rem;display:flex;gap:1.25rem;font-size:0.8rem;color:#64748b">
          <span>Runs: <strong style="color:#e2e8f0">{{ p.total_runs }}</strong></span>
          <span style="color:#4ade80">Pass: {{ p.passed_runs }}</span>
          <span style="color:#f87171">Fail: {{ p.failed_runs }}</span>
        </div>
        <div style="margin-top:0.5rem;font-size:0.75rem;color:#475569">
          Last run: {{ formatDate(p.last_run) }}
        </div>
      </RouterLink>
    </div>
  </template>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { fetchJSON, formatDate, statusClass, type ProjectSummary } from '../types'

const projects = ref<ProjectSummary[]>([])
const loading = ref(true)
const error = ref('')

const passingCount = computed(() => projects.value.filter(p => p.latest_status === 'success').length)
const failingCount = computed(() => projects.value.filter(p => p.latest_status !== 'success').length)

onMounted(async () => {
  try {
    projects.value = await fetchJSON<ProjectSummary[]>('/api/projects')
  } catch (e) {
    error.value = 'Failed to load projects.'
  } finally {
    loading.value = false
  }
})
</script>
