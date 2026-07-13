// loader.c
#include "loader.h"
#include "libretro.h"
#include <dlfcn.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

static const char* SYSTEM_DIR = "./";
static const char* SAVE_DIR = "./";

// 内部结构体，缓存所有 dlsym 获得的函数指针
typedef struct {
    void* handle;

    void (*retro_init)(void);
    void (*retro_deinit)(void);
    bool (*retro_load_game)(const struct retro_game_info* game);
    void (*retro_run)(void);
    void (*retro_reset)(void);
    void (*retro_get_system_info)(struct retro_system_info*);
    void (*retro_get_system_av_info)(struct retro_system_av_info*);
    unsigned (*retro_api_version)(void);
    void (*retro_set_environment)(retro_environment_t);
    void (*retro_set_video_refresh)(retro_video_refresh_t);
    void (*retro_set_audio_sample)(retro_audio_sample_t);
    void (*retro_set_audio_sample_batch)(retro_audio_sample_batch_t);
    void (*retro_set_input_poll)(retro_input_poll_t);
    void (*retro_set_input_state)(retro_input_state_t);
    size_t (*retro_serialize_size)(void);
    bool (*retro_serialize)(void*, size_t);
    bool (*retro_unserialize)(const void*, size_t);
    void* (*retro_get_memory_data)(unsigned);
    size_t (*retro_get_memory_size)(unsigned);
    void (*retro_set_controller_port_device)(unsigned, unsigned);
} core_t;

// 全局单例 — 单进程仅运行一个 libretro 内核实例
static core_t* g_core;

// 像素格式：由 RETRO_ENVIRONMENT_SET_PIXEL_FORMAT 回调设置，-1 表示未设置
static int g_pixel_format = -1;

// Go 导出的回调函数（单例模式，无需 corePtr 参数）
extern void goVideoRefreshCB(void* data, unsigned int width, unsigned int height, size_t pitch);
extern void goAudioSampleCB(int16_t left, int16_t right);
extern size_t goAudioSampleBatchCB(void* data, size_t frames);
extern void goInputPollCB(void);
extern int16_t goInputStateCB(unsigned int port, unsigned int device, unsigned int index, unsigned int id);

// C 包装回调 — 直接转发到 Go 导出函数

static void c_video_refresh(const void* data, unsigned int width, unsigned int height, size_t pitch) {
    goVideoRefreshCB((void*)data, width, height, pitch);
}

static void c_audio_sample(int16_t left, int16_t right) {
    goAudioSampleCB(left, right);
}

static size_t c_audio_sample_batch(const int16_t* data, size_t frames) {
    return goAudioSampleBatchCB((void*)data, frames);
}

static void c_input_poll(void) {
    goInputPollCB();
}

static int16_t c_input_state(unsigned port, unsigned device, unsigned index, unsigned id) {
    return goInputStateCB(port, device, index, id);
}

// 辅助宏：dlsym 并赋值
#define LOAD_SYM(core, name) \
    core->name = dlsym(core->handle, #name); \
    if (!core->name) { dlclose(core->handle); free(core); return NULL; }

void* core_load(const char* path) {
    void* h = dlopen(path, RTLD_NOW | RTLD_GLOBAL);
    if (!h) { return NULL;}

    core_t* c = calloc(1, sizeof(core_t));
    if (!c) { dlclose(h); return NULL; }
    c->handle = h;

    // 必须符号
    LOAD_SYM(c, retro_init);
    LOAD_SYM(c, retro_deinit);
    LOAD_SYM(c, retro_load_game);
    LOAD_SYM(c, retro_run);
    LOAD_SYM(c, retro_get_system_info);
    LOAD_SYM(c, retro_get_system_av_info);
    LOAD_SYM(c, retro_api_version);
    LOAD_SYM(c, retro_set_environment);
    LOAD_SYM(c, retro_set_video_refresh);
    LOAD_SYM(c, retro_set_audio_sample);
    LOAD_SYM(c, retro_set_audio_sample_batch);
    LOAD_SYM(c, retro_set_input_poll);
    LOAD_SYM(c, retro_set_input_state);

    // 可选符号（可能为 NULL，不致命）
    c->retro_reset = dlsym(h, "retro_reset");
    c->retro_serialize_size = dlsym(h, "retro_serialize_size");
    c->retro_serialize = dlsym(h, "retro_serialize");
    c->retro_unserialize = dlsym(h, "retro_unserialize");
    c->retro_get_memory_data = dlsym(h, "retro_get_memory_data");
    c->retro_get_memory_size = dlsym(h, "retro_get_memory_size");
    c->retro_set_controller_port_device = dlsym(h, "retro_set_controller_port_device");

    return c;
}

void core_unload(void* core) {
    core_t* c = (core_t*)core;
    if (c) { dlclose(c->handle); free(c); }
}

bool core_environment_cb(unsigned int cmd, void* data) {
    switch(cmd) {
    case RETRO_ENVIRONMENT_GET_SYSTEM_DIRECTORY:
        printf("cmd: GET_SYSTEM_DIRECTORY, return: %s\n", SYSTEM_DIR);
        *(const char**)data = SYSTEM_DIR;
        return true;
    case RETRO_ENVIRONMENT_GET_SAVE_DIRECTORY:
        printf("cmd: GET_SAVE_DIRECTORY, return: %s\n", SYSTEM_DIR);
        *(const char**)data = SAVE_DIR;
        return true;
    case RETRO_ENVIRONMENT_SET_PIXEL_FORMAT: {
        const enum retro_pixel_format *fmt = (const enum retro_pixel_format*)data;
        g_pixel_format = (int)*fmt;
        printf("set pixel format: %d (0=0RGB1555, 1=XRGB8888, 2=RGB565)\n", g_pixel_format);
        return true;
    }
    default:
        // Unknown command - don't pretend to handle it
        return false;
    }
}

void core_init(void* core) {
    g_core = (core_t*)core;
    ((core_t*)core)->retro_init();
}

void core_deinit(void* core)           { ((core_t*)core)->retro_deinit(); }

void core_run(void* core) {
    g_core = (core_t*)core;
    ((core_t*)core)->retro_run();
}

void core_reset(void* core)            { core_t* c = (core_t*)core; if (c->retro_reset) c->retro_reset(); }
unsigned core_api_version(void* core)   { return ((core_t*)core)->retro_api_version(); }

void core_set_environment(void* core) { 
    printf("core set environment\n"); 
    ((core_t*)core)->retro_set_environment(core_environment_cb); 
}

// 回调注册 — 全部使用 C 静态包装函数，直接转发到 Go 单例
void core_set_video_refresh(void* core)
    { printf("core set video refresh\n"); ((core_t*)core)->retro_set_video_refresh(c_video_refresh); }
void core_set_audio_sample(void *core) {
    printf("core set audio sample\n");
    ((core_t*)core)->retro_set_audio_sample(c_audio_sample);
}
void core_set_audio_sample_batch(void* core)
    { printf("core set audio sample batch\n"); ((core_t*)core)->retro_set_audio_sample_batch(c_audio_sample_batch); }
void core_set_input_poll(void* core)
    { printf("core set input poll\n"); ((core_t*)core)->retro_set_input_poll(c_input_poll); }
void core_set_input_state(void* core)
    { printf("core set input state\n"); ((core_t*)core)->retro_set_input_state(c_input_state); }


bool core_load_game_file(void* core, const char* rom_path) {
    core_t* c = (core_t*)core;
    struct retro_game_info info;
    memset(&info, 0, sizeof(info));
    info.path = rom_path;
    info.data = 0;
    info.size = 0;
    printf("loading game file: %s\n", rom_path);
    return c->retro_load_game(&info);
}

bool core_load_game_data(void* core, const char* rom_path, void* data, size_t size) {
    core_t* c = (core_t*)core;
    struct retro_game_info info;
    memset(&info, 0, sizeof(info));
    info.path = rom_path;
    info.data = data;
    info.size = size;
    printf("loading game data, path: %s, size: %ld\n", rom_path, size);
    return c->retro_load_game(&info);
}

void core_get_system_info(void* core, struct retro_system_info* info)
    { ((core_t*)core)->retro_get_system_info(info); }
void core_get_system_av_info(void* core, struct retro_system_av_info* info)
    { ((core_t*)core)->retro_get_system_av_info(info); }

size_t core_serialize_size(void* core) {
    core_t* c = (core_t*)core;
    return (c->retro_serialize_size) ? c->retro_serialize_size() : 0;
}
bool core_serialize(void* core, void* data, size_t size) {
    core_t* c = (core_t*)core;
    return (c->retro_serialize) ? c->retro_serialize(data, size) : false;
}
bool core_unserialize(void* core, const void* data, size_t size) {
    core_t* c = (core_t*)core;
    return (c->retro_unserialize) ? c->retro_unserialize(data, size) : false;
}
void* core_get_memory_data(void* core, unsigned id) {
    core_t* c = (core_t*)core;
    return (c->retro_get_memory_data) ? c->retro_get_memory_data(id) : NULL;
}
size_t core_get_memory_size(void* core, unsigned id) {
    core_t* c = (core_t*)core;
    return (c->retro_get_memory_size) ? c->retro_get_memory_size(id) : 0;
}
void core_set_controller_port_device(void* core, unsigned port, unsigned device) {
    core_t* c = (core_t*)core;
    if (c->retro_set_controller_port_device) c->retro_set_controller_port_device(port, device);
}

int core_get_pixel_format(void* core) {
    (void)core;
    return g_pixel_format;
}
