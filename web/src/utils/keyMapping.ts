// 手柄按键映射工具
// 维护按键名 → 键盘键的映射，并提供按键名 → libretro JOYPAD bit 位的查询

// 按键名（与 KeyMappingDialog 中的 buttons 配置对齐）
export type ButtonName =
  | 'B'
  | 'Y'
  | 'Select'
  | 'Start'
  | 'Up'
  | 'Down'
  | 'Left'
  | 'Right'
  | 'A'
  | 'X'
  | 'L'
  | 'R'
  | 'TurboA'
  | 'TurboB'

/**
 * 按键名 → buttons uint16 中的 bit 位
 * 0..11 对齐 libretro RETRO_DEVICE_ID_JOYPAD_*：
 *   B=0, Y=1, Select=2, Start=3, Up=4, Down=5, Left=6, Right=7, A=8, X=9, L=10, R=11
 * 12, 13 为本项目自定义连发位，EmuRunner 暂未使用，前端发送即可
 */
export const BUTTON_BITS: Record<ButtonName, number> = {
  B: 0,
  Y: 1,
  Select: 2,
  Start: 3,
  Up: 4,
  Down: 5,
  Left: 6,
  Right: 7,
  A: 8,
  X: 9,
  L: 10,
  R: 11,
  TurboA: 12,
  TurboB: 13,
}

/** 默认键盘映射 */
export const DEFAULT_KEY_MAPPING: Record<ButtonName, string> = {
  A: 'KeyZ',
  B: 'KeyX',
  Y: '',
  X: '',
  Start: 'Enter',
  Select: 'ShiftLeft',
  Up: 'ArrowUp',
  Down: 'ArrowDown',
  Left: 'ArrowLeft',
  Right: 'ArrowRight',
  L: 'KeyQ',
  R: 'KeyE',
  TurboA: '',
  TurboB: '',
}

const STORAGE_KEY = 'cloudemu_key_mapping'

/** 从 localStorage 读取按键映射，未设置或损坏时返回默认映射 */
export function loadMapping(): Record<ButtonName, string> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...DEFAULT_KEY_MAPPING }
    const parsed = JSON.parse(raw) as Partial<Record<ButtonName, string>>
    return { ...DEFAULT_KEY_MAPPING, ...parsed }
  } catch {
    return { ...DEFAULT_KEY_MAPPING }
  }
}

/** 保存按键映射到 localStorage */
export function saveMapping(mapping: Record<ButtonName, string>): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(mapping))
  } catch {
    // localStorage 不可用时静默失败
  }
}

/**
 * 反向构建：键盘键名 → ButtonName，便于 keydown/keyup 快速查找
 * 同一键盘键映射到多个 ButtonName 时取最后一个（实际不应有这种情况）
 */
export function buildReverseMap(
  mapping: Record<ButtonName, string>,
): Record<string, ButtonName> {
  const reverse: Record<string, ButtonName> = {}
  for (const [btn, key] of Object.entries(mapping) as [ButtonName, string][]) {
    if (key) reverse[key] = btn
  }
  return reverse
}