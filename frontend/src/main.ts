import 'element-plus/dist/index.css'
import './style.css'

import ElementPlus from 'element-plus'
import { createPinia } from 'pinia'
import { createApp } from 'vue'

import App from './App.vue'
import { loadRuntimeData } from './api/runtimeData'
import router from './router'

await loadRuntimeData()

createApp(App).use(createPinia()).use(router).use(ElementPlus).mount('#app')
