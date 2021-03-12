package streamer

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/gotk3/gotk3/glib"
	"github.com/swampapp/swamp/internal/eventbus"
	"github.com/swampapp/swamp/internal/index"
	"github.com/swampapp/swamp/internal/logger"
)

var (
	StreamingStarted = "streamer.starteds"
	StreamingStopped = "streamer.stopped"
)

func init() {
	eventbus.RegisterEvents(StreamingStarted, StreamingStopped)
}

func Stream(fileID string) error {
	idx, err := index.Client()
	if err != nil {
		logger.Error(err, "error initializing the index")
		return err
	}

	logger.Print("streaming ", fileID)

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
			eventbus.Emit(context.Background(), StreamingStarted, nil)
		})

		defer glib.IdleAdd(func() {
			eventbus.Emit(context.Background(), StreamingStopped, nil)
		})

		err = idx.Fetch(ctx, fileID, stdin)
		if err != nil && err != context.Canceled {
			logger.Error(err, "error streaming file")
			return
		}
		logger.Info("streaming finished")
	}()

	err = cmd.Run()
	if err != nil {
		logger.Errorf(err, "error playing file %s", fileID)
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
