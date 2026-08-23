package main

import (
	"context"
	_ "embed"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/leonst036/NetConnect/daemon"
	"github.com/leonst036/NetConnect/utils"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func runDaemon() {
	if os.Geteuid() != 0 {
		fmt.Println("Notice: NetConnect Daemon is running without root. For TUN interface creation, run with sudo/pkexec.")
	}

	relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:4535")
	targetID := utils.GetEnv("NETLINK_TARGET_ID", "")

	srv := daemon.NewDaemonServer(relayURL, targetID)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(4545); err != nil {
			fmt.Printf("[NetConnect Daemon] Server error: %v\n", err)
		}
	}()

	fmt.Println("[NetConnect Daemon] Control server listening on http://127.0.0.1:4545")
	<-sigChan
	fmt.Println("\n[NetConnect Daemon] Shutting down...")
	srv.Stop()
}

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--daemon" || arg == "daemon" {
			runDaemon()
			return
		}
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "NetConnect",
		Width:             480,
		Height:            600,
		MinWidth:          480,
		MinHeight:         600,
		MaxWidth:          480,
		MaxHeight:         600,
		DisableResize:     true,
		AlwaysOnTop:       false,
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			go setupTray(app, ctx)
		},
		OnShutdown: func(ctx context.Context) {
			app.shutdown(ctx)
		},
		Linux: &linux.Options{
			Icon:             appIcon,
			ProgramName:      "netconnect",
			WebviewGpuPolicy: linux.WebviewGpuPolicyNever,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "c87df35b-netconnect-gui",
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				if app.ctx != nil {
					runtime.WindowShow(app.ctx)
					runtime.WindowUnminimise(app.ctx)
				}
			},
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

