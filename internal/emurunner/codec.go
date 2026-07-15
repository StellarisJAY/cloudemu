package emurunner

import (
	"image"
	"io"
	"log/slog"
	"time"

	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/mediadevices/pkg/codec/opus"
	"github.com/pion/mediadevices/pkg/codec/x264"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/mediadevices/pkg/wave"
	"github.com/pion/webrtc/v4/pkg/media"
)

type FrameReader struct {
	frame *image.YCbCr
	w     int
	h     int
}

func (f *FrameReader) Read() (img image.Image, release func(), err error) {
	img = f.frame
	release = func() {}
	return
}

func (f *FrameReader) SetFrame(frame *image.YCbCr, w, h int) {
	f.frame = frame
}

type X264Encoder struct {
	params *x264.Params
	reader *FrameReader
	enc    codec.ReadCloser
}

func NewX264Encoder() *X264Encoder {
	reader := &FrameReader{}
	return &X264Encoder{
		reader: reader,
	}
}

func (e *X264Encoder) Init(w, h int) error {
	params, err := x264.NewParams()
	if err != nil {
		return err
	}
	media := prop.Media{
		DeviceID: "cloudemu-video",
		Video: prop.Video{
			Width:       w,
			Height:      h,
			FrameRate:   60,
			FrameFormat: frame.FormatI420,
		},
	}
	params.BitRate = 800_000
	params.KeyFrameInterval = 20
	params.Preset = x264.PresetUltrafast
	e.reader.w = w
	e.reader.h = h
	e.params = &params
	enc, err := e.params.BuildVideoEncoder(e.reader, media)
	if err != nil {
		return err
	}
	e.enc = enc
	return nil
}

func (e *X264Encoder) Encode(frame *image.YCbCr, w, h int) (media.Sample, error) {
	e.reader.SetFrame(frame, w, h)
	data, _, err := e.enc.Read()
	sample := media.Sample{}
	if err != nil {
		return sample, err
	}
	sample.Data = data
	sample.Duration = time.Millisecond * 15
	sample.Timestamp = time.Now()
	return sample, nil
}

// AudioBufferReader 通过 channel 桥接 libretro 音频回调和 opus 编码器
// 实现 pion/mediadevices 的 audio.Reader 接口
type AudioBufferReader struct {
	chunks chan wave.Audio
}

func NewAudioBufferReader() *AudioBufferReader {
	return &AudioBufferReader{
		chunks: make(chan wave.Audio, 4),
	}
}

func (r *AudioBufferReader) Read() (wave.Audio, func(), error) {
	chunk, ok := <-r.chunks
	if !ok {
		return nil, func() {}, io.EOF
	}
	return chunk, func() {}, nil
}

func (r *AudioBufferReader) PushChunk(audio wave.Audio) {
	r.chunks <- audio
}

// OpusEncoder 使用 pion/mediadevices 软件 Opus 编码器，镜像 X264Encoder 模式
type OpusEncoder struct {
	params *opus.Params
	reader *AudioBufferReader
	enc    codec.ReadCloser
}

func NewOpusEncoder() *OpusEncoder {
	return &OpusEncoder{
		reader: NewAudioBufferReader(),
	}
}

// Init 初始化 Opus 编码器
// sampleRate: libretro 采样率（通常 48000 Hz）
// channels: 声道数，1=NES 单声道，2=GB/DOS 立体声
func (e *OpusEncoder) Init(sampleRate, channels int) error {
	params, err := opus.NewParams()
	if err != nil {
		return err
	}
	params.BitRate = 96000
	params.Latency = opus.Latency20ms

	media := prop.Media{
		DeviceID: "cloudemu-audio",
		Audio: prop.Audio{
			ChannelCount: channels,
			SampleRate:   sampleRate,
			Latency:      time.Duration(opus.Latency20ms),
		},
	}
	slog.Info("create opus encoder", "sampleRate", sampleRate, "channels", channels)
	e.params = &params
	enc, err := params.BuildAudioEncoder(e.reader, media)
	if err != nil {
		return err
	}
	e.enc = enc
	return nil
}

// Encode 将交错 int16 PCM 数据编码为 Opus media.Sample
// pcm: 交错 int16 PCM 样本（长度 = 帧大小 = sampleRate * 20ms / 1000 * channels）
// sampleRate: 采样率
// channels: 声道数
func (e *OpusEncoder) Encode(pcm []int16, sampleRate, channels int) (media.Sample, error) {
	chunk := &wave.Int16Interleaved{
		Data: pcm,
		Size: wave.ChunkInfo{
			Len:          len(pcm) / channels,
			Channels:     channels,
			SamplingRate: sampleRate,
		},
	}
	e.reader.PushChunk(chunk)

	data, _, err := e.enc.Read()
	sample := media.Sample{}
	if err != nil {
		return sample, err
	}
	sample.Data = data
	sample.Duration = time.Millisecond * 20
	sample.Timestamp = time.Now()
	return sample, nil
}
