<template>
  <h1>Deployments</h1>
  <DeploymentTabs />

  <div v-if="loading" class="loading">Loading deployments...</div>
  <div v-else-if="error" class="loading" role="alert">{{ error }}</div>
  <div v-else-if="overview.projects.length === 0" class="loading">
    No deployments yet. Send a deployment webhook to get started.
  </div>
  <template v-else>
    <p class="deployment-note">Latest successful deployment per environment. Hover over a date for the exact local time.</p>
    <div class="grid deployment-grid">
      <article
        v-for="project in projects"
        :key="project.project"
        class="card deployment-card"
      >
        <RouterLink :to="`/deployments/project/${encodeURIComponent(project.project)}`" class="deployment-project-link">
          <h2 class="deployment-project-name">{{ project.project }}</h2>
        </RouterLink>
        <dl class="deployment-environments">
          <div v-for="deployment in project.environments" :key="deployment.environment" class="deployment-environment">
            <dt class="environment-name">
              <RouterLink :to="{ path: `/deployments/project/${encodeURIComponent(project.project)}`, query: { environment: deployment.environment } }"
                class="deployment-environment-link">{{ deployment.environment }}</RouterLink>
            </dt>
            <dd>
              <VersionLink v-if="deployment.version || deployment.version_created_at" :deployment="deployment" fallback="Version"
                class="deployment-version deployment-summary-link" />
              <RouterLink :to="`/deployments/project/${encodeURIComponent(project.project)}`" class="deployment-summary-link">
                <span class="deployment-deployed-date">Deployed <RelativeDate :value="deployment.deployed_at" /></span>
              </RouterLink>
            </dd>
            <dd v-if="deployment.deployment_url" class="deployment-environment-actions">
              <EnvironmentActions :project="project.project" :environment="deployment.environment" :deployment-url="deployment.deployment_url" />
            </dd>
          </div>
        </dl>
        <DeploymentCompare :project="project.project" :environments="project.environments" />
      </article>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { fetchJSON, type DeploymentOverview } from '../types'
import RelativeDate from '../components/RelativeDate.vue'
import DeploymentTabs from '../components/DeploymentTabs.vue'
import EnvironmentActions from '../components/EnvironmentActions.vue'
import VersionLink from '../components/VersionLink.vue'
import DeploymentCompare from '../components/DeploymentCompare.vue'

const overview = ref<DeploymentOverview>({ projects: [] })
const loading = ref(true)
const error = ref('')
const projects = computed(() => overview.value.projects.map(project => ({
  project: project.project,
  environments: [...project.environments].sort((a, b) =>
    Date.parse(b.deployed_at) - Date.parse(a.deployed_at) || a.environment.localeCompare(b.environment)
  ),
})))

onMounted(async () => {
  try {
    overview.value = await fetchJSON<DeploymentOverview>('/api/deployments/projects')
  } catch {
    error.value = 'Failed to load deployments.'
  } finally {
    loading.value = false
  }
})
</script>
