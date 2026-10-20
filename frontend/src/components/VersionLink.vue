<template>
  <a :href="deployment.git_url" target="_blank" rel="noopener noreferrer" class="ext" :title="tooltip">
    {{ deployment.version || fallback }}<span v-if="deployment.version_created_at" class="deployment-version-age"> (created <RelativeDate :value="deployment.version_created_at" :tooltip="tooltip" />)</span>
  </a>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Deployment } from '../types'
import { formatVersionTooltip } from '../dates'
import RelativeDate from './RelativeDate.vue'

const props = withDefaults(defineProps<{ deployment: Deployment; fallback?: string }>(), {
  fallback: 'View commit',
})
const tooltip = computed(() => formatVersionTooltip(
  props.deployment.version_created_at, props.deployment.deployed_at, props.deployment.git_url,
))
</script>
