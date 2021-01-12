package streamer

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/gotk3/gotk3/glib"
	"github.com/rs/zerolog/log"
	"github.com/swampapp/swamp/internal/index"
)

func Stream(fileID string) error {
	idx, err := index.Client()
	if err != nil {
		log.Error().Err(err).Msg("error initializing the index")
		return err
	}

	log.Print("streaming ", fileID)

	cmd, err := findPlayer()
	if err != nil {
		return err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		glib.IdleAdd(func() {
			if onStartStreaming != nil {
				onStartStreaming()
			}
			//status.StartDownloading()
		})
		defer glib.IdleAdd(func() {
			if onStopStreaming != nil {
				onStopStreaming()
			}
			//status.StopDownloading()
		})
		err = idx.Fetch(ctx, fileID, stdin)
		if err != nil && err != context.Canceled {
			log.Error().Err(err).Msg("error streaming file")
			return
		}
		log.Info().Msg("streaming finished")
	}()

	err = cmd.Run()
	if err != nil {
		log.Error().Err(err).Msgf("error playing file %s", fileID)
		return err
	}
	return err
}

func findPlayer() (*exec.Cmd, error) {
	cmd := exec.Command("mpv", "--player-operation-mode=pseudo-gui", "--force-window", "-")
	_, err := exec.LookPath("mpv")
	if err != nil {
		_, err := exec.LookPath("vlc")
		if err != nil {
			return cmd, fmt.Errorf("mpv or vlc not found in PATH")
		}
		cmd = exec.Command("vlc", "-")
	}

	return cmd, nil
}

var onStartStreaming func()
var onStopStreaming func()

func OnStartStreaming(fn func()) {
	onStartStreaming = fn
}

func OnStopStreaming(fn func()) {
	onStopStreaming = fn
}
