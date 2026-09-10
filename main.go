package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/leonst036/NetConnect/daemon"
	"github.com/leonst036/NetConnect/utils"
)

func main() {
	if os.Geteuid() != 0 {
		fmt.Println("Notice: NetConnect Daemon is running without root. For TUN interface creation, run with sudo.")
	}

	relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:4535")
	targetID := utils.GetEnv("NETLINK_TARGET_ID", "")

	srv := daemon.NewDaemonServer(relayURL, targetID)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	portStr := utils.GetEnv("NETCONNECT_PORT", "4545")
	port := 4545
	if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
		port = p
	}

	go func() {
		if err := srv.Start(port); err != nil {
			fmt.Printf("[NetConnect Daemon] Server error: %v\n", err)
		}
	}()

	fmt.Println("[NetConnect Daemon] Ready. Start GUI (`wails dev`) as normal user to connect.")
	<-sigChan
	fmt.Println("\n[NetConnect Daemon] Shutting down...")
	srv.Stop()
}
