<template>
  <RouterLink to="/deployments" class="back">&#8592; Back to deployments</RouterLink>
  <h1 class="project-name">{{ projectName }} — Deployments</h1>
  <p class="deployment-note">Successful deployments, newest first. Hover over a date for the exact local time.</p>
  <DeploymentHistory :project="projectName" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import DeploymentHistory from '../components/DeploymentHistory.vue'

const route = useRoute()
// Vue Router already decodes route parameters.
const projectName = computed(() => route.params.name as string)
</script>
