import { ref, onUnmounted, watch, type Ref } from 'vue'
import {
  BUTTON_BITS,
  buildReverseMap,
  loadMapping,
  type ButtonName,
} from '@/utils/keyMapping'

/**
 * 输入数据包格式 (4 bytes)：
 *   [0] = 0x01      type prefix（input）
 *   [1] = buttons_lo (uint16 little-endian)
 *   [2] = buttons_hi
 *   [3] = 0          保留字节
 */
const PACKET_TYPE_INPUT = 0x01

/**
 * useGameInput
 * 监听键盘事件，按 60Hz 通过 publishInput 发送当前按键状态到 EmuRunner
 *
 * @param publishInput LiveKit DataChannel 发送函数，由 useLiveKit 提供
 * @param enabled 是否启用输入采集，未启用时不监听键盘也不发送
 */
export function useGameInput(
  publishInput: (data: Uint8Array, topic: string) => void,
  enabled: Ref<boolean>,
) {
  // 当前按键状态 (uint16 bitset)，bit i 对应 BUTTON_BITS 中的某个按钮
  let state = 0
  // 当前键盘映射（按 KeyboardEvent.code 索引）
  const mapping = ref(loadMapping())
  const reverseMap = ref(buildReverseMap(mapping.value))

  /** 重新加载映射（KeyMappingDialog 保存后调用） */
  function reloadMapping() {
    mapping.value = loadMapping()
    reverseMap.value = buildReverseMap(mapping.value)
  }

  function applyButton(btn: ButtonName, pressed: boolean) {
    const bit = BUTTON_BITS[btn]
    if (pressed) {
      state |= 1 << bit
    } else {
      state &= ~(1 << bit) & 0xffff
    }
  }

  function onKeyDown(e: KeyboardEvent) {
    const btn = reverseMap.value[e.code]
    if (!btn) return
    // 防止页面滚动等默认行为（方向键、空格等）
    e.preventDefault()
    if (e.repeat) return
    applyButton(btn, true)
  }

  function onKeyUp(e: KeyboardEvent) {
    const btn = reverseMap.value[e.code]
    if (!btn) return
    e.preventDefault()
    applyButton(btn, false)
  }

  /** 失焦时释放所有按键，避免长按"卡键" */
  function onBlur() {
    state = 0
  }

  /** 释放所有按键，供虚拟手柄等外部调用方使用 */
  function releaseAll() {
    state = 0
  }

  // RAF 循环：每帧发送当前状态
  let rafId: number | null = null
  let lastTick = 0
  const TICK_MS = 1000 / 60 // 60Hz

  function loop(ts: number) {
    if (!enabled.value) {
      rafId = null
      return
    }
    if (ts - lastTick >= TICK_MS) {
      lastTick = ts
      // 始终发送（unreliable 通道，最新覆盖；接收端按 bit 解析）
      const packet = new Uint8Array(4)
      packet[0] = PACKET_TYPE_INPUT
      packet[1] = state & 0xff
      packet[2] = (state >> 8) & 0xff
      packet[3] = 0
      publishInput(packet, 'input')
    }
    rafId = requestAnimationFrame(loop)
  }

  function start() {
    if (rafId !== null) return
    state = 0
    window.addEventListener('keydown', onKeyDown)
    window.addEventListener('keyup', onKeyUp)
    window.addEventListener('blur', onBlur)
    rafId = requestAnimationFrame(loop)
  }

  function stop() {
    if (rafId !== null) {
      cancelAnimationFrame(rafId)
      rafId = null
    }
    window.removeEventListener('keydown', onKeyDown)
    window.removeEventListener('keyup', onKeyUp)
    window.removeEventListener('blur', onBlur)
    state = 0
  }

  // 根据 enabled 自动启停
  watch(
    enabled,
    (on) => {
      if (on) {
        start()
      } else {
        stop()
      }
    },
    { immediate: true },
  )

  onUnmounted(() => {
    stop()
  })

  return {
    applyButton,
    releaseAll,
    reloadMapping,
  }
}