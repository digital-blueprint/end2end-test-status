<template>
  <div class="dashboard-tabs" role="navigation" aria-label="E2E dashboard sections">
    <RouterLink to="/">Overview</RouterLink>
    <RouterLink to="/results">All Results</RouterLink>
  </div>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>
