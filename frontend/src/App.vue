<template>
  <nav>
    <span class="brand">E2E Test Status</span>
    <RouterLink to="/">Dashboard</RouterLink>
    <RouterLink to="/results">All Results</RouterLink>
    <button
      class="theme-toggle"
      type="button"
      :aria-pressed="theme === 'dark'"
      :title="theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'"
      @click="toggleTheme"
    >
      {{ theme === 'dark' ? 'Light mode' : 'Dark mode' }}
    </button>
  </nav>
  <main>
    <RouterView />
  </main>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, RouterView } from 'vue-router'

type Theme = 'light' | 'dark'

const theme = ref<Theme>((document.documentElement.dataset.theme as Theme) || 'dark')

const applyTheme = (nextTheme: Theme) => {
  theme.value = nextTheme
  document.documentElement.dataset.theme = nextTheme
  localStorage.setItem('theme', nextTheme)
}

const toggleTheme = () => {
  applyTheme(theme.value === 'dark' ? 'light' : 'dark')
}
</script>
