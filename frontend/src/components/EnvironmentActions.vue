<template>
  <div class="deployment-actions">
    <a v-if="deploymentUrl" :href="deploymentUrl" target="_blank" rel="noopener noreferrer"
      class="deployment-action-button" :title="deploymentUrl"
      :aria-label="`Open ${project} environment ${environment}`">Open environment &#8599;</a>
  </div>
</template>

<script setup lang="ts">
defineProps<{ project: string; environment: string; deploymentUrl: string }>()

</script>
