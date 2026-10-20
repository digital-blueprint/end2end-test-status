<template>
  <h1>Deployments</h1>
  <DeploymentTabs />
  <p class="deployment-note">All successful deployment reports, newest first. Hover over a date for the exact local time.</p>
  <DeploymentHistory />
</template>

<script setup lang="ts">
import DeploymentTabs from '../components/DeploymentTabs.vue'
import DeploymentHistory from '../components/DeploymentHistory.vue'
</script>
