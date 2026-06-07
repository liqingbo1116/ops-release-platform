import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'

import PermissionButton from './PermissionButton.vue'
import { useAuthStore } from '@/stores/authStore'

describe('PermissionButton', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('enables button when current user has permission', () => {
    localStorage.setItem('ops-release-token', 'mock-token-admin')
    const pinia = createPinia()
    setActivePinia(pinia)
    useAuthStore()

    const wrapper = mount(PermissionButton, {
      props: { permission: 'user:write' },
      slots: { default: '新增用户' },
      global: {
        plugins: [pinia],
        stubs: {
          ElButton: {
            props: ['disabled', 'type'],
            template: '<button :disabled="disabled"><slot /></button>',
          },
        },
      },
    })

    expect(wrapper.find('button').attributes('disabled')).toBeUndefined()
  })

  it('disables button when permission is missing', () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const wrapper = mount(PermissionButton, {
      props: { permission: 'user:write' },
      slots: { default: '新增用户' },
      global: {
        plugins: [pinia],
        stubs: {
          ElButton: {
            props: ['disabled', 'type'],
            template: '<button :disabled="disabled"><slot /></button>',
          },
        },
      },
    })

    expect(wrapper.find('button').attributes('disabled')).toBeDefined()
  })
})
