// loader.h
#ifndef LOADER_H
#define LOADER_H

#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>
#include "libretro.h"

// 加载/卸载内核
void*       core_load(const char* path);
void        core_unload(void* handle);

// 生命周期函数
void        core_init(void* handle);
void        core_deinit(void* handle);
bool        core_load_game_file(void* handle, const char* rom_path);
bool        core_load_game_data(void* core, const char* rom_path, void* data, size_t size);
void        core_unload_game(void* handle);
void        core_run(void* handle);
void        core_reset(void* handle);

// 信息查询
void        core_get_system_info(void* handle, struct retro_system_info* info);
void        core_get_system_av_info(void* handle, struct retro_system_av_info* info);
unsigned    core_api_version(void* handle);

// 回调注册（内部使用 C 包装函数，直接转发到 Go 单例）
void        core_set_environment(void* handle);
void        core_set_video_refresh(void* handle);
void        core_set_audio_sample(void* handle);
void        core_set_audio_sample_batch(void* handle);
void        core_set_input_poll(void* handle);
void        core_set_input_state(void* handle);
bool        core_environment_cb(unsigned int cmd, void* data);

// 存档
size_t      core_serialize_size(void* handle);
bool        core_serialize(void* handle, void* data, size_t size);
bool        core_unserialize(void* handle, const void* data, size_t size);

// 内存访问
void*       core_get_memory_data(void* handle, unsigned id);
size_t      core_get_memory_size(void* handle, unsigned id);

// 端口设备
void        core_set_controller_port_device(void* handle, unsigned port, unsigned device);

// 像素格式（由 RETRO_ENVIRONMENT_SET_PIXEL_FORMAT 回调设置）
int         core_get_pixel_format(void* handle);

#endif
