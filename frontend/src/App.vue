<template>
  <nav>
    <span class="brand">Project Status</span>
    <RouterLink to="/" :class="{ 'router-link-active': !route.path.startsWith('/deployments') }">E2E Tests</RouterLink>
    <RouterLink to="/deployments" :class="{ 'router-link-active': route.path.startsWith('/deployments') }">Deployments</RouterLink>
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
  <footer v-if="footerText" class="app-footer">
    <span>{{ footerText }}</span>
  </footer>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { footerText } from './types'

type Theme = 'light' | 'dark'
const route = useRoute()

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
