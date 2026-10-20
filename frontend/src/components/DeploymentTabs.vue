<template>
  <div class="dashboard-tabs" role="navigation" aria-label="Deployment sections">
    <RouterLink to="/deployments">Overview</RouterLink>
    <RouterLink to="/deployments/events">All Deployments</RouterLink>
  </div>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>
