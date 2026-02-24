import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './style.css'

type Theme = 'light' | 'dark'

const savedTheme = localStorage.getItem('theme') as Theme | null
const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
const theme = savedTheme || (prefersDark ? 'dark' : 'light')
document.documentElement.dataset.theme = theme

const app = createApp(App)
app.use(router)
app.mount('#app')
