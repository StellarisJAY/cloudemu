import { ref, computed, watchEffect } from 'vue'

const THEME_KEY = 'cloudemu-theme'

type Theme = 'dark' | 'light'

const theme = ref<Theme>((localStorage.getItem(THEME_KEY) as Theme) || 'dark')

watchEffect(() => {
  document.documentElement.setAttribute('data-theme', theme.value)
  localStorage.setItem(THEME_KEY, theme.value)
})

export function useTheme() {
  const isDark = computed(() => theme.value === 'dark')

  function toggle() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
  }

  return { theme, isDark, toggle }
}
