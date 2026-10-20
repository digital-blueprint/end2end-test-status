<template>
  <time :datetime="datetime" :title="tooltip ?? exact" :aria-label="`${relative} (${exact})`">{{ relative }}</time>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatDate, formatRelativeDate, parseDate } from '../dates'
import { useNow } from '../useNow'

const props = defineProps<{ value: string; tooltip?: string }>()
const now = useNow()
const exact = computed(() => formatDate(props.value))
const relative = computed(() => formatRelativeDate(props.value, now.value))
const datetime = computed(() => {
  const date = parseDate(props.value)
  return Number.isFinite(date.getTime()) ? date.toISOString() : undefined
})
</script>
