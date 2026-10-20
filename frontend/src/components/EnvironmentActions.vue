<template>
  <details class="deployment-actions" @keydown.esc="close" @focusout="closeOnFocusOut">
    <summary :aria-label="`Actions for ${environment}`" :title="`Actions for ${environment}`">&#8943;</summary>
    <div class="deployment-action-menu">
      <a v-if="deploymentUrl" :href="deploymentUrl" target="_blank" rel="noopener noreferrer"
        :title="deploymentUrl" @click="close">Open environment &#8599;</a>
      <RouterLink :to="historyRoute" @click="close">See deployments</RouterLink>
    </div>
  </details>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'

const props = defineProps<{ project: string; environment: string; deploymentUrl: string }>()
const historyRoute = computed(() => ({
  path: `/deployments/project/${encodeURIComponent(props.project)}`,
  query: { environment: props.environment },
}))

function close(event: Event) {
  const details = (event.currentTarget as HTMLElement).closest('details')
  if (details) {
    details.open = false
    if (event.type === 'keydown') details.querySelector('summary')?.focus()
  }
}

function closeOnFocusOut(event: FocusEvent) {
  const details = event.currentTarget as HTMLDetailsElement
  if (!details.contains(event.relatedTarget as Node | null)) details.open = false
}
</script>
