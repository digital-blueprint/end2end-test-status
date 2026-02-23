<template>
  <nav>
    <span class="brand">E2E Test Status</span>
    <RouterLink to="/">Dashboard</RouterLink>
    <RouterLink to="/results">All Results</RouterLink>
  </nav>
  <main>
    <RouterView />
  </main>
</template>

<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
</script>
