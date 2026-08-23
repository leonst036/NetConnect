package utils

import (
	"os"
	"os/exec"
)


// GetEnv retrieves the value of the environment variable named by the key, or returns the fallback.
func GetEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}


// NotifyUser sends a desktop notification to the user, handling sudo sessions if applicable.
func NotifyUser(title, message string) error {
	sudoUser := os.Getenv("SUDO_USER")
	sudoUID := os.Getenv("SUDO_UID")

	var cmd *exec.Cmd
	if sudoUser != "" && sudoUID != "" {
		cmd = exec.Command("sudo", "-u", sudoUser, "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/"+sudoUID+"/bus", "notify-send", title, message)
	} else {
		cmd = exec.Command("notify-send", title, message)
	}

	return cmd.Run()
}
