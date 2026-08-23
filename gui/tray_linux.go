//go:build linux

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type TrayNotifier struct {
	app *App
	ctx context.Context
}

func (t *TrayNotifier) Activate(x, y int32) *dbus.Error {
	if t.ctx != nil {
		runtime.WindowShow(t.ctx)
		runtime.WindowUnminimise(t.ctx)
	}
	return nil
}

func (t *TrayNotifier) SecondaryActivate(x, y int32) *dbus.Error {
	if t.ctx != nil {
		runtime.WindowShow(t.ctx)
		runtime.WindowUnminimise(t.ctx)
	}
	return nil
}

func (t *TrayNotifier) ContextMenu(x, y int32) *dbus.Error {
	if t.ctx != nil {
		runtime.WindowShow(t.ctx)
		runtime.WindowUnminimise(t.ctx)
	}
	return nil
}

func (t *TrayNotifier) Scroll(delta int32, orientation string) *dbus.Error {
	return nil
}

func setupTray(app *App, ctx context.Context) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Printf("[NetConnect Tray] Failed to connect to Session Bus: %v\n", err)
		return
	}

	serviceName := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := conn.RequestName(serviceName, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		fmt.Printf("[NetConnect Tray] Failed to request name %s: %v\n", serviceName, err)
		return
	}

	notifier := &TrayNotifier{app: app, ctx: ctx}
	err = conn.Export(notifier, "/StatusNotifierItem", "org.kde.StatusNotifierItem")
	if err != nil {
		fmt.Printf("[NetConnect Tray] Failed to export notifier: %v\n", err)
		return
	}

	propsSpec := prop.Map{
		"org.kde.StatusNotifierItem": {
			"Category": {
				Value: "ApplicationStatus",
			},
			"Id": {
				Value: "netconnect",
			},
			"Title": {
				Value: "NetConnect",
			},
			"Status": {
				Value: "Active",
			},
			"IconName": {
				Value: "network-vpn",
			},
			"IconThemePath": {
				Value: "",
			},
			"ItemIsMenu": {
				Value: false,
			},
		},
	}

	_, err = prop.Export(conn, "/StatusNotifierItem", propsSpec)
	if err != nil {
		fmt.Printf("[NetConnect Tray] Failed to export props: %v\n", err)
		return
	}

	watcherObj := conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher")
	call := watcherObj.Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, serviceName)
	if call.Err != nil {
		_ = watcherObj.Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, "/StatusNotifierItem")
	}

	fmt.Println("[NetConnect Tray] StatusNotifierItem registered successfully on desktop tray.")
}
