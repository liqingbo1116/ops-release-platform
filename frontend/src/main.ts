import './style.css'

import { createPinia } from 'pinia'
import { createApp } from 'vue'

import App from './App.vue'
import { loadRuntimeData } from './api/runtimeData'
import router from './router'

await loadRuntimeData()

createApp(App).use(createPinia()).use(router).mount('#app')
