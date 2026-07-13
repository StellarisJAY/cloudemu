package emurunner

import (
	"image"
	"time"

	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/mediadevices/pkg/codec/x264"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/prop"
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
