package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

func initializeXCLip() error {
	if runtime.GOOS == "windows" {
		return nil
	}

	for i := 1; i <= 5; i++ {
		if _, xclipErr := exec.Command("xclip").Output(); xclipErr != nil {
			if i >= 5 {
				return fmt.Errorf("Похоже, что XClip не установлен: %s", xclipErr.Error())
			}

			time.Sleep(time.Second * 5)
		} else {
			break
		}
	}

	return nil
}
