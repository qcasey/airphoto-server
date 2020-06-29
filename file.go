package main

import (
	"fmt"
	"os"
	"os/exec"
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// SaveFrame writes a video's thumbnail frame to disk
func SaveFrame(width int, height int, videoPath string, outputImagePath string) error {
	cmd := exec.Command("ffmpeg", "-i", videoPath, "-vframes", "1", "-s", fmt.Sprintf("%dx%d", width, height), "-f", "singlejpeg", fmt.Sprintf(outputImagePath))
	if cmd.Run() != nil {
		return fmt.Errorf("could not generate frame")
	}

	return nil
}
