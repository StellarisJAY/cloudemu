<script setup lang="ts">
import { darkTheme } from 'naive-ui'
import { darkThemeOverrides, lightThemeOverrides } from '@/styles/naive-overrides'
import { useTheme } from '@/composables/useTheme'

const { isDark, toggle } = useTheme()
</script>

<template>
  <n-config-provider
    :theme="isDark ? darkTheme : null"
    :theme-overrides="isDark ? darkThemeOverrides : lightThemeOverrides"
  >
    <n-loading-bar-provider>
      <n-notification-provider>
        <n-message-provider>
          <n-dialog-provider>
            <router-view />
          </n-dialog-provider>

          <!-- 主题切换浮动按钮 -->
          <button
            class="theme-toggle"
            :title="isDark ? '切换亮色主题' : '切换暗色主题'"
            @click="toggle"
          >
            <!-- 太阳图标 -->
            <svg
              v-if="isDark"
              viewBox="0 0 24 24"
              width="20"
              height="20"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <circle cx="12" cy="12" r="5" />
              <line x1="12" y1="1" x2="12" y2="3" />
              <line x1="12" y1="21" x2="12" y2="23" />
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
              <line x1="1" y1="12" x2="3" y2="12" />
              <line x1="21" y1="12" x2="23" y2="12" />
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
            </svg>
            <!-- 月亮图标 -->
            <svg
              v-else
              viewBox="0 0 24 24"
              width="20"
              height="20"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
            </svg>
          </button>
        </n-message-provider>
      </n-notification-provider>
    </n-loading-bar-provider>
  </n-config-provider>
</template>

<style>
/* 全局浮动主题切换按钮 */
.theme-toggle {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: 1000;
  width: 42px;
  height: 42px;
  border: 1px solid var(--color-border);
  border-radius: 50%;
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  backdrop-filter: blur(8px);
  box-shadow: var(--shadow-md);
  transition:
    transform 0.2s,
    color 0.2s,
    background 0.2s;
}

.theme-toggle:hover {
  color: var(--color-accent);
  transform: scale(1.1);
}

.theme-toggle:active {
  transform: scale(0.95);
}
</style>
