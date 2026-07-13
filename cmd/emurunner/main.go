package main

import (
	"context"
	"flag"

	"github.com/StellarisJAY/cloudemu/internal/emurunner"
	"github.com/StellarisJAY/cloudemu/internal/emurunner/backend"
)

var (
	PublisherHost = flag.String("publisher-host", "localhost:7899", "livekit host url")
	Token         = flag.String("token", "", "livekit room token")
	RoomID        = flag.String("room", "", "room ID")
	ROM           = flag.String("rom", "", "rom file path")
	BackendType   = flag.String("backend", "nes", "emulator backend type: \"nes\",\"gb\"")
	HostIdentity  = flag.String("host-identity", "", "host player livekit identity, e.g. player:{host_user_id}, default bound to port 0")
	Upscale       = flag.Bool("upscale", true, "enable integer nearest-neighbor upscale to preserve pixel sharpness")
)

func main() {
	flag.Parse()
	if *Token == "" {
		panic("missing room token")
	}
	if *RoomID == "" {
		panic("missing room id")
	}
	if *BackendType == "" {
		panic("missing emulator backend type")
	}
	if *ROM == "" {
		panic("missing rom path")
	}
	if *HostIdentity == "" {
		panic("missing host identity")
	}
	config := emurunner.LiveKitConfig{
		HostURL: *PublisherHost,
		Token:   *Token,
		RoomID:  *RoomID,
	}
	instance := emurunner.NewInstance(config, backend.Type(*BackendType), *HostIdentity, *Upscale)
	if err := instance.InitRunner(*ROM); err != nil {
		panic(err)
	}
	if err := instance.InitPublisher(); err != nil {
		panic(err)
	}
	instance.Run(context.Background())
}
