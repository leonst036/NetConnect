package main

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"image/png"
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

type Pixmap struct {
	Width  int32
	Height int32
	Data   []byte
}

func getIconPixmaps() []Pixmap {
	img, err := png.Decode(bytes.NewReader(appIcon))
	if err != nil {
		return nil
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	data := make([]byte, w*h*4)
	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			data[idx] = c.A
			data[idx+1] = c.R
			data[idx+2] = c.G
			data[idx+3] = c.B
			idx += 4
		}
	}
	return []Pixmap{{Width: int32(w), Height: int32(h), Data: data}}
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
				Value: "netconnect",
			},
			"IconPixmap": {
				Value: getIconPixmaps(),
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
