package backend

/*
#cgo LDFLAGS: -ldl
#include "loader.h"
#include "libretro.h"

extern void goVideoRefreshCB(void* data, unsigned int width, unsigned int height, size_t pitch);
extern void goAudioSampleCB(int16_t left, int16_t right);
extern size_t goAudioSampleBatchCB(void* data, size_t frames);
extern void goInputPollCB(void);
extern int16_t goInputStateCB(unsigned int port, unsigned int device, unsigned int index, unsigned int id);
*/
import "C"

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"unsafe"
)

// 单例路由 — 单进程仅运行一个 libretro 内核实例
var currentBackend *LibretroBackend

// YCbCr 颜色转换预计算查找表（消除热点循环中的位运算和 color.RGBToYCbCr 函数调用开销）
// 各表覆盖对应像素格式全部 65536 种 16-bit 颜色，每项 3 字节 (Y, Cb, Cr)
var rgb565ToYCbCrLut   [65536][3]byte
var rgb1555ToYCbCrLut  [65536][3]byte

func init() {
	// RGB565: R5G6B5, little-endian 拼接为 uint16 后 [R4..R0 G5..G0 B4..B0]
	for i := 0; i < 65536; i++ {
		r5 := (i >> 11) & 0x1F
		g6 := (i >> 5) & 0x3F
		b5 := i & 0x1F
		r := byte(uint(r5) * 255 / 31)
		g := byte(uint(g6) * 255 / 63)
		b := byte(uint(b5) * 255 / 31)
		rgb565ToYCbCrLut[i][0], rgb565ToYCbCrLut[i][1], rgb565ToYCbCrLut[i][2] = color.RGBToYCbCr(r, g, b)
	}
	// 0RGB1555: A1R5G5B5, 忽略 bit15 (A)
	for i := 0; i < 65536; i++ {
		r5 := (i >> 10) & 0x1F
		g5 := (i >> 5) & 0x1F
		b5 := i & 0x1F
		r := byte(uint(r5) * 255 / 31)
		g := byte(uint(g5) * 255 / 31)
		b := byte(uint(b5) * 255 / 31)
		rgb1555ToYCbCrLut[i][0], rgb1555ToYCbCrLut[i][1], rgb1555ToYCbCrLut[i][2] = color.RGBToYCbCr(r, g, b)
	}
}

type RGBFrame []byte

type Type string

const (
	BackEndTypeNES Type = "nes"
	BackendTypeGB  Type = "gb"
	BackendTypeDOS Type = "dos"
)

// 像素格式常量（对应 libretro enum retro_pixel_format）
const (
	PixelFormat0RGB1555 = 0 // 已废弃的默认格式，2 字节/像素
	PixelFormatXRGB8888 = 1 // XRGB8888，4 字节/像素
	PixelFormatRGB565   = 2 // RGB565，2 字节/像素
)

// LibretroBackend 每个实例对应一个独立的 libretro 内核（dlopen 加载的 .so）
type LibretroBackend struct {
	ptr             unsafe.Pointer // C 层 core_t* 指针
	SystemInfo      LibretroSystemInfo
	AVInfo          LibretroSystemAVInfo
	PixelFormat     int // libretro 像素格式（0=0RGB1555, 1=XRGB8888, 2=RGB565）
	firstFrame      bool
	romData         []byte
	frameOutputChan chan *image.YCbCr

	buffers      [2]*image.YCbCr
	activeBuffer int

	lastFrame *image.YCbCr

	inputProvider InputProvider // 玩家输入查询接口，nil 时所有按钮始终返回 0

	ScaleFactor int // 整数倍 nearest-neighbor 放大系数，1 表示不放大

	// 音频 PCM 缓冲与通道
	audioBuf       []int16 // PCM 累积缓冲区
	audioChan      chan []int16 // 满一帧后发送到此通道
	audioFrameSize int // 每个 Opus 帧应累积的 int16 样本数（sampleRate * latencyMs / 1000 * channels）
	audioChannels  int // 声道数（1=单声道，2=立体声）
}

// InputProvider 玩家输入查询接口，由上层 InputManager 实现
// 在 libretro retro_input_state 回调中查询当前按键状态
type InputProvider interface {
	// GetButton 返回指定端口上某个按钮的按下状态（1=按下，0=释放）
	// port: 模拟器手柄端口号（0-based）
	// id: libretro RETRO_DEVICE_ID_JOYPAD_* 常量
	GetButton(port int, id int) int16
}

type LibretroSystemInfo struct {
	needFullPath bool
	Name         string
	Version      string
}

type LibretroSystemAVInfo struct {
	BaseWidth  int
	BaseHeight int
	MaxWidth   int
	MaxHeight  int
	SampleRate float64
	FPS        float64
}

// NewBackend 创建新的 LibretroBackend 实例（替代原单例 GetInstance）
func NewBackend() *LibretroBackend {
	return &LibretroBackend{
		frameOutputChan: make(chan *image.YCbCr, 1),
		audioChan:       make(chan []int16, 8),
		firstFrame:      true,
		ScaleFactor:     1, // 默认不放大，由上层按需设置
	}
}

// ScaleFactorForType 按模拟器类型返回推荐的整数倍 nearest-neighbor 放大系数
// 确保输出分辨率接近 720p~1080p，保留像素颗粒的锐利边缘
func ScaleFactorForType(t Type) int {
	switch t {
	case BackEndTypeNES:
		return 3 // 256×240 → 768×720（720p）
	case BackendTypeGB:
		return 4 // 240×160 → 960×640
	case BackendTypeDOS:
		return 2 // 640×480 → 1280×960
	default:
		return 3
	}
}

// SetScaleFactor 设置整数倍 nearest-neighbor 放大系数，必须在第一帧视频回调之前调用
func (lb *LibretroBackend) SetScaleFactor(factor int) {
	lb.ScaleFactor = factor
}

// LoadBackend 从so文件加载libretro模拟器后端，设置为全局单例
func (lb *LibretroBackend) LoadBackend(path string) error {
	cpath := C.CString(path)

	lb.ptr = C.core_load(cpath)
	if lb.ptr == nil {
		return fmt.Errorf("failed to load core: %s", path)
	}

	currentBackend = lb

	return nil
}

// UnloadBackend 卸载内核并清除单例
func (lb *LibretroBackend) UnloadBackend() {
	if lb.ptr == nil {
		return
	}
	currentBackend = nil

	C.core_unload(lb.ptr)
	lb.ptr = nil
}

func (lb *LibretroBackend) Init() {
	C.core_set_environment(lb.ptr)
	C.core_init(lb.ptr)
	lb.RegisterCallbacks()
	lb.GetSystemInfo()
}

func (lb *LibretroBackend) Deinit() {
	C.core_deinit(lb.ptr)
}

func (lb *LibretroBackend) Run() {
	C.core_run(lb.ptr)
}

// Serialize 序列化当前模拟器运行状态（存档）
// 调用 retro_serialize_size 获取所需缓冲区大小，再调用 retro_serialize 写入
func (lb *LibretroBackend) Serialize() ([]byte, error) {
	size := C.core_serialize_size(lb.ptr)
	if size == 0 {
		return nil, errors.New("serialize not supported by core")
	}
	buf := make([]byte, int(size))
	if !bool(C.core_serialize(lb.ptr, unsafe.Pointer(&buf[0]), size)) {
		return nil, errors.New("retro_serialize failed")
	}
	return buf, nil
}

// Unserialize 从存档数据恢复模拟器运行状态（读档）
func (lb *LibretroBackend) Unserialize(data []byte) error {
	if len(data) == 0 {
		return errors.New("empty save state data")
	}
	if !bool(C.core_unserialize(lb.ptr, unsafe.Pointer(&data[0]), C.size_t(len(data)))) {
		return errors.New("retro_unserialize failed")
	}
	return nil
}

// LoadGameFile 加载rom文件
func (lb *LibretroBackend) LoadGameFile(path string) bool {
	cpath := C.CString(path)
	if lb.SystemInfo.needFullPath {
		return bool(C.core_load_game_file(lb.ptr, cpath))
	} else {
		file, err := os.Open(path)
		if err != nil {
			return false
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			return false
		}
		lb.romData = data

		return bool(C.core_load_game_data(lb.ptr, cpath, unsafe.Pointer(&lb.romData[0]), C.size_t(len(data))))
	}
}

func (lb *LibretroBackend) GetSystemInfo() {
	var sysInfo C.struct_retro_system_info
	C.core_get_system_info(lb.ptr, &sysInfo)
	lb.SystemInfo.Name = C.GoString(sysInfo.library_name)
	lb.SystemInfo.Version = C.GoString(sysInfo.library_version)
	lb.SystemInfo.needFullPath = bool(sysInfo.need_fullpath)
}

// GetPixelFormat 从 C 层获取 core 请求的像素格式
func (lb *LibretroBackend) GetPixelFormat() {
	lb.PixelFormat = int(C.core_get_pixel_format(lb.ptr))
}

func (lb *LibretroBackend) GetSysAVInfo() {
	var avInfo C.struct_retro_system_av_info
	C.core_get_system_av_info(lb.ptr, &avInfo)
	lb.AVInfo.BaseWidth = int(avInfo.geometry.base_width)
	lb.AVInfo.BaseHeight = int(avInfo.geometry.base_height)
	lb.AVInfo.MaxWidth = int(avInfo.geometry.max_width)
	lb.AVInfo.MaxHeight = int(avInfo.geometry.max_height)
	lb.AVInfo.FPS = float64(avInfo.timing.fps)
	lb.AVInfo.SampleRate = float64(avInfo.timing.sample_rate)
}

func (lb *LibretroBackend) FrameChan() chan *image.YCbCr {
	return lb.frameOutputChan
}

// AudioChan 返回音频输出通道，每次发送一个 Opus 帧所需 PCM 数据（[]int16 交错样本）
func (lb *LibretroBackend) AudioChan() chan []int16 {
	return lb.audioChan
}

// SetAudioConfig 配置音频参数（必须在 Init() 之后、Run() 之前调用）
// sampleRate: 采样率（Hz），从 retro_system_av_info 获取，通常 48000
// channels: 声道数，1=单声道（NES），2=立体声（GB/DOS）
// latencyMs: Opus 帧时长（毫秒），默认 20
func (lb *LibretroBackend) SetAudioConfig(sampleRate float64, channels int, latencyMs int) {
	lb.audioChannels = channels
	lb.audioFrameSize = int(sampleRate) * latencyMs * channels / 1000
}

// SetInputProvider 注入玩家输入查询接口
// 必须在 Init() 之后、Run() 之前调用，避免与 libretro 回调并发
func (lb *LibretroBackend) SetInputProvider(p InputProvider) {
	lb.inputProvider = p
}

func (lb *LibretroBackend) SwitchBuffer() *image.YCbCr {
	if lb.activeBuffer == 0 {
		lb.activeBuffer = 1
	} else {
		lb.activeBuffer = 0
	}
	return lb.buffers[lb.activeBuffer]
}

// RegisterCallbacks 注册所有 libretro 回调
// C 层使用内部包装函数，直接转发到 Go 单例，无需传入 Go 函数指针
func (lb *LibretroBackend) RegisterCallbacks() {
	C.core_set_video_refresh(lb.ptr)
	C.core_set_audio_sample(lb.ptr)
	C.core_set_audio_sample_batch(lb.ptr)
	C.core_set_input_poll(lb.ptr)
	C.core_set_input_state(lb.ptr)
}

// ==============================
// Go 导出的 libretro 回调函数（单例模式，直接使用 currentBackend）
// ==============================

// readRGB 从原始帧缓冲区读取 RGB 像素值，处理三种 libretro 像素格式
func readRGB(raw []byte, src int, pixelFormat int) (r, g, b byte) {
	switch pixelFormat {
	case PixelFormatXRGB8888:
		// XRGB8888 little-endian 内存布局: [B, G, R, X]
		r = raw[src+2] // R
		g = raw[src+1] // G
		b = raw[src+0] // B
	case PixelFormatRGB565:
		// RGB565 little-endian: byte0=GGGBBBBB, byte1=RRRRRGGG
		lo, hi := raw[src], raw[src+1]
		r5 := (hi >> 3) & 0x1F
		g6 := ((hi & 0x07) << 3) | ((lo >> 5) & 0x07)
		b5 := lo & 0x1F
		r = byte(uint(r5) * 255 / 31) // R
		g = byte(uint(g6) * 255 / 63) // G
		b = byte(uint(b5) * 255 / 31) // B
	case PixelFormat0RGB1555:
		// 0RGB1555 little-endian: byte0=GGGBBBBB, byte1=ARRRRRGG
		lo, hi := raw[src], raw[src+1]
		r5 := (hi >> 2) & 0x1F
		g5 := ((hi & 0x03) << 3) | ((lo >> 5) & 0x07)
		b5 := lo & 0x1F
		r = byte(uint(r5) * 255 / 31) // R
		g = byte(uint(g5) * 255 / 31) // G
		b = byte(uint(b5) * 255 / 31) // B
	default:
		r = raw[src+2]
		g = raw[src+1]
		b = raw[src+0]
	}
	return
}

//export goVideoRefreshCB
func goVideoRefreshCB(data unsafe.Pointer, width C.uint, height C.uint, cpitch C.size_t) {
	lb := currentBackend
	if lb == nil {
		return
	}

	w, h, pitch := int(width), int(height), int(cpitch)
	scale := lb.ScaleFactor
	sw, sh := w*scale, h*scale

	if lb.buffers[0] == nil {
		lb.buffers[0] = image.NewYCbCr(image.Rect(0, 0, sw, sh), image.YCbCrSubsampleRatio420)
		lb.buffers[1] = image.NewYCbCr(image.Rect(0, 0, sw, sh), image.YCbCrSubsampleRatio420)
		lb.activeBuffer = 0
	}

	totalBytes := int(pitch * h)
	// 当前循环没有画面输出，直接输出上一帧
	if data == nil {
		if lb.lastFrame != nil {
			lb.frameOutputChan <- lb.lastFrame
		}
		return
	}
	raw := C.GoBytes(data, C.int(totalBytes))

	var bytesPerPixel int
	switch lb.PixelFormat {
	case PixelFormatXRGB8888:
		bytesPerPixel = 4
	case PixelFormatRGB565, PixelFormat0RGB1555:
		bytesPerPixel = 2
	default:
		bytesPerPixel = 4
	}

	ycbcr := lb.SwitchBuffer()

	// 预判是否可用 LUT（2 字节像素格式），并选定对应查找表；XRGB8888 走原始 readRGB + RGBToYCbCr 路径
	var ycLut *[65536][3]byte
	useLUT := lb.PixelFormat == PixelFormatRGB565 || lb.PixelFormat == PixelFormat0RGB1555
	if useLUT {
		if lb.PixelFormat == PixelFormat0RGB1555 {
			ycLut = &rgb1555ToYCbCrLut
		} else {
			ycLut = &rgb565ToYCbCrLut
		}
	}

	if useLUT {
		if scale == 1 {
			// LUT + 无放大
			for y := range h {
				for x := range w {
					src := y*pitch + x*bytesPerPixel
					color16 := uint16(raw[src]) | (uint16(raw[src+1]) << 8)
					yc := ycLut[color16]
					ycbcr.Y[ycbcr.YOffset(x, y)] = yc[0]
					ycbcr.Cb[ycbcr.COffset(x, y)] = yc[1]
					ycbcr.Cr[ycbcr.COffset(x, y)] = yc[2]
				}
			}
		} else {
			// LUT + 整数倍 nearest-neighbor 放大
			for y := range h {
				baseOy := y * scale
				for x := range w {
					src := y*pitch + x*bytesPerPixel
					baseOx := x * scale
					color16 := uint16(raw[src]) | (uint16(raw[src+1]) << 8)
					yc := ycLut[color16]

					for dy := 0; dy < scale; dy++ {
						rowStart := ycbcr.YOffset(baseOx, baseOy+dy)
						for dx := 0; dx < scale; dx++ {
							ycbcr.Y[rowStart+dx] = yc[0]
						}
					}
					for dy := 0; dy < scale; dy += 2 {
						for dx := 0; dx < scale; dx += 2 {
							co := ycbcr.COffset(baseOx+dx, baseOy+dy)
							ycbcr.Cb[co] = yc[1]
							ycbcr.Cr[co] = yc[2]
						}
					}
				}
			}
		}
	} else {
		if scale == 1 {
			// XRGB8888 + 无放大
			for y := range h {
				for x := range w {
					src := y*pitch + x*bytesPerPixel
					r, g, b := readRGB(raw, src, lb.PixelFormat)
					Y, Cb, Cr := color.RGBToYCbCr(r, g, b)
					ycbcr.Y[ycbcr.YOffset(x, y)] = Y
					ycbcr.Cb[ycbcr.COffset(x, y)] = Cb
					ycbcr.Cr[ycbcr.COffset(x, y)] = Cr
				}
			}
		} else {
			// XRGB8888 + 整数倍 nearest-neighbor 放大
			for y := range h {
				baseOy := y * scale
				for x := range w {
					src := y*pitch + x*bytesPerPixel
					baseOx := x * scale
					r, g, b := readRGB(raw, src, lb.PixelFormat)
					Y, Cb, Cr := color.RGBToYCbCr(r, g, b)

					for dy := 0; dy < scale; dy++ {
						rowStart := ycbcr.YOffset(baseOx, baseOy+dy)
						for dx := 0; dx < scale; dx++ {
							ycbcr.Y[rowStart+dx] = Y
						}
					}
					for dy := 0; dy < scale; dy += 2 {
						for dx := 0; dx < scale; dx += 2 {
							co := ycbcr.COffset(baseOx+dx, baseOy+dy)
							ycbcr.Cb[co] = Cb
							ycbcr.Cr[co] = Cr
						}
					}
				}
			}
		}
	}

	lb.lastFrame = ycbcr
	lb.frameOutputChan <- ycbcr
}

//export goAudioSampleCB
func goAudioSampleCB(left C.int16_t, right C.int16_t) {
	_ = currentBackend
	// TODO
}

//export goAudioSampleBatchCB
func goAudioSampleBatchCB(data unsafe.Pointer, frames C.size_t) C.size_t {
	lb := currentBackend
	if lb == nil || lb.audioChan == nil || lb.audioFrameSize == 0 {
		return frames
	}
	channels := lb.audioChannels
	if channels == 0 {
		channels = 2 // 默认立体声
	}
	n := int(frames) * channels
	if n == 0 {
		return frames
	}
	// 复制 C 内存中的 int16 PCM 数据到 Go 切片
	pcm := unsafe.Slice((*int16)(data), n)
	for _, v := range pcm {
		lb.audioBuf = append(lb.audioBuf, v)
	}
	// 累积满一帧后发送到通道（非阻塞，避免阻塞 C 回调）
	for len(lb.audioBuf) >= lb.audioFrameSize {
		chunk := make([]int16, lb.audioFrameSize)
		copy(chunk, lb.audioBuf[:lb.audioFrameSize])
		lb.audioBuf = lb.audioBuf[lb.audioFrameSize:]
		select {
		case lb.audioChan <- chunk:
		default:
			// 通道满则丢弃，防止 C 回调阻塞
		}
	}
	return frames
}

//export goInputPollCB
func goInputPollCB() {
	_ = currentBackend
	// TODO
}

//export goInputStateCB
func goInputStateCB(port C.uint, device C.uint, index C.uint, id C.uint) C.int16_t {
	lb := currentBackend
	if lb == nil {
		return 0
	}
	// 只处理 JOYPAD 设备，其他设备（鼠标等）返回 0
	if device != 1 { // RETRO_DEVICE_JOYPAD = 1
		return 0
	}
	if lb.inputProvider == nil {
		return 0
	}
	return C.int16_t(lb.inputProvider.GetButton(int(port), int(id)))
}

func GetBackendFilePath(backendType Type) (string, error) {
	switch backendType {
	case BackEndTypeNES:
		return "./libretro/quicknes_libretro.so", nil
	case BackendTypeGB:
		return "./libretro/mgba_libretro.so", nil
	case BackendTypeDOS:
		return "./libretro/dosbox_libretro.so", nil
	default:
		return "", errors.New("unsupported backend")
	}
}
