<template>
  <section v-if="environments.length > 1" class="deployment-compare" :aria-label="`Compare ${project} environments`">
    <div class="filters deployment-compare-controls">
      <select :id="`${id}-base`" v-model="baseEnvironment" aria-label="Old environment" title="Old environment">
        <option v-for="deployment in environments" :key="deployment.environment" :value="deployment.environment">
          {{ deployment.environment }}
        </option>
      </select>
      <button type="button" class="deployment-action-button" @click="swapEnvironments"
        title="Swap old and new environments" :aria-label="`Swap old and new environments for ${project}`">
        &#8644;
      </button>
      <select :id="`${id}-target`" v-model="targetEnvironment" aria-label="New environment" title="New environment">
        <option v-for="deployment in environments" :key="deployment.environment" :value="deployment.environment">
          {{ deployment.environment }}
        </option>
      </select>
      <a v-if="compareURL" :href="compareURL" target="_blank" rel="noopener noreferrer"
        class="deployment-action-button" :title="compareURL"
        :aria-label="`Compare ${project}: ${baseEnvironment} to ${targetEnvironment} (opens in a new tab)`">
        Compare &#8599;
      </a>
      <button v-else type="button" class="deployment-action-button" disabled :aria-describedby="`${id}-hint`">Compare &#8599;</button>
    </div>
    <p :id="`${id}-hint`" class="deployment-compare-hint">
      {{ hint }}
    </p>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import type { Deployment } from '../types'
import { buildDeploymentCompareURL, hasSameDeploymentRevision } from '../deploymentCompare'

const props = defineProps<{ project: string; environments: Deployment[] }>()
const id = useId()
const baseEnvironment = ref('')
const targetEnvironment = ref('')

function swapEnvironments() {
  [baseEnvironment.value, targetEnvironment.value] = [targetEnvironment.value, baseEnvironment.value]
}

watch(() => props.environments, environments => {
  const names = environments.map(deployment => deployment.environment)
  if (!names.includes(baseEnvironment.value)) {
    baseEnvironment.value = names[names.length - 1] ?? ''
  }
  if (!names.includes(targetEnvironment.value)) {
    targetEnvironment.value = names[0] ?? ''
  }
}, { immediate: true })

const baseDeployment = computed(() => props.environments.find(deployment => deployment.environment === baseEnvironment.value))
const targetDeployment = computed(() => props.environments.find(deployment => deployment.environment === targetEnvironment.value))
const sameRevision = computed(() => {
  const base = baseDeployment.value
  const target = targetDeployment.value
  return !!base && !!target && hasSameDeploymentRevision(base.git_url, target.git_url)
})
const compareURL = computed(() => {
  if (baseEnvironment.value === targetEnvironment.value) return null
  const base = baseDeployment.value
  const target = targetDeployment.value
  return base && target ? buildDeploymentCompareURL(base.git_url, target.git_url) : null
})
const hint = computed(() => {
  if (baseEnvironment.value === targetEnvironment.value) return 'Select two different environments.'
  if (sameRevision.value) return 'Both environments have the same deployed version. No changes to compare.'
  if (!compareURL.value) return 'Requires GitHub or GitLab commit links from the same repository.'
  return `See what changed from ${baseEnvironment.value} (left) to ${targetEnvironment.value} (right), using their latest deployed versions. Use ⇄ to reverse the comparison.`
})
</script>
