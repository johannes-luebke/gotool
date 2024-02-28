package notify

import (
	"fmt"
	"log"
	"os/exec"
)

func NotifyOS(title string, message string) {
	// TODO make this OS independent
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`display dialog "%s" with title "%s" with icon caution buttons {"OK"} default button "OK"`, message, "Mapps - "+title))
	err := cmd.Run()
	if err != nil {
		log.Println(err)
	}
}
