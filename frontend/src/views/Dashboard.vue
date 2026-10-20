<template>
  <h1>E2E Tests</h1>
  <E2EDashboardTabs />

  <div v-if="loading" class="loading">Loading projects...</div>
  <div v-else-if="error" class="loading">{{ error }}</div>
  <div v-else-if="projects.length === 0" class="loading">
    No projects yet. Send a webhook to get started.
  </div>
  <template v-else>
    <div class="stats">
      <button type="button" class="stat-box clickable" :class="{ active: filter === 'all' }" :aria-pressed="filter === 'all'" @click="setFilter('all')">
        <div class="label">Projects</div>
        <div class="value">{{ projects.length }}</div>
      </button>
      <button type="button" class="stat-box clickable" :class="{ active: filter === 'passing' }" :aria-pressed="filter === 'passing'" @click="setFilter('passing')">
        <div class="label">Passing</div>
        <div class="value" style="color:#4ade80">{{ passingCount }}</div>
      </button>
      <button type="button" class="stat-box clickable" :class="{ active: filter === 'failing' }" :aria-pressed="filter === 'failing'" @click="setFilter('failing')">
        <div class="label">Failing</div>
        <div class="value" style="color:#f87171">{{ failingCount }}</div>
      </button>
    </div>

    <div v-if="filteredProjects.length === 0" class="loading">
      No {{ filter }} projects.
    </div>
    <div v-else class="grid">
      <RouterLink
        v-for="p in filteredProjects"
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
          Last run: <RelativeDate :value="p.last_run" />
        </div>
      </RouterLink>
    </div>
  </template>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import E2EDashboardTabs from '../components/E2EDashboardTabs.vue'
import { fetchJSON, statusClass, type ProjectSummary } from '../types'
import RelativeDate from '../components/RelativeDate.vue'

type Filter = 'all' | 'passing' | 'failing'

const route = useRoute()
const router = useRouter()

const projects = ref<ProjectSummary[]>([])
const loading = ref(true)
const error = ref('')

const isPassing = (p: ProjectSummary) => p.latest_status === 'success'

const passingCount = computed(() => projects.value.filter(isPassing).length)
const failingCount = computed(() => projects.value.filter(p => !isPassing(p)).length)

const filter = computed<Filter>(() => {
  const s = route.query.status
  return s === 'passing' || s === 'failing' ? s : 'all'
})

const filteredProjects = computed(() => {
  if (filter.value === 'passing') return projects.value.filter(isPassing)
  if (filter.value === 'failing') return projects.value.filter(p => !isPassing(p))
  return projects.value
})

function setFilter(f: Filter) {
  const next = f === filter.value ? 'all' : f
  router.replace({ query: { ...route.query, status: next === 'all' ? undefined : next } })
}

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
