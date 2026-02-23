import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import ProjectView from '../views/ProjectView.vue'
import AllResults from '../views/AllResults.vue'

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Dashboard },
    { path: '/project/:name', component: ProjectView },
    { path: '/results', component: AllResults },
  ],
})
