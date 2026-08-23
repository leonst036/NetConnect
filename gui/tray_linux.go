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

type DBusMenu struct {
	app *App
	ctx context.Context
}

type MenuLayout struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}

type MenuProps struct {
	ID         int32
	Properties map[string]dbus.Variant
}

type MenuEvent struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

func (m *DBusMenu) GetLayout(parentId int32, recursionDepth int32, propertyNames []string) (uint32, MenuLayout, *dbus.Error) {
	item1 := MenuLayout{
		ID: 1,
		Properties: map[string]dbus.Variant{
			"label":   dbus.MakeVariant("Show NetConnect"),
			"enabled": dbus.MakeVariant(true),
			"visible": dbus.MakeVariant(true),
		},
		Children: []dbus.Variant{},
	}

	item2 := MenuLayout{
		ID: 2,
		Properties: map[string]dbus.Variant{
			"label":   dbus.MakeVariant("Quit"),
			"enabled": dbus.MakeVariant(true),
			"visible": dbus.MakeVariant(true),
		},
		Children: []dbus.Variant{},
	}

	root := MenuLayout{
		ID:         0,
		Properties: map[string]dbus.Variant{},
		Children: []dbus.Variant{
			dbus.MakeVariant(item1),
			dbus.MakeVariant(item2),
		},
	}

	return 1, root, nil
}

func (m *DBusMenu) GetGroupProperties(ids []int32, propertyNames []string) ([]MenuProps, *dbus.Error) {
	var res []MenuProps
	for _, id := range ids {
		props := map[string]dbus.Variant{
			"enabled": dbus.MakeVariant(true),
			"visible": dbus.MakeVariant(true),
		}
		if id == 1 {
			props["label"] = dbus.MakeVariant("Show NetConnect")
		} else if id == 2 {
			props["label"] = dbus.MakeVariant("Quit")
		}
		res = append(res, MenuProps{ID: id, Properties: props})
	}
	return res, nil
}

func (m *DBusMenu) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	if name == "label" {
		if id == 1 {
			return dbus.MakeVariant("Show NetConnect"), nil
		}
		if id == 2 {
			return dbus.MakeVariant("Quit"), nil
		}
	}
	if name == "enabled" || name == "visible" {
		return dbus.MakeVariant(true), nil
	}
	return dbus.MakeVariant(""), nil
}

func (m *DBusMenu) Event(id int32, eventId string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if eventId == "clicked" {
		m.handleClick(id)
	}
	return nil
}

func (m *DBusMenu) EventGroup(events []MenuEvent) ([]int32, *dbus.Error) {
	for _, ev := range events {
		if ev.EventID == "clicked" {
			m.handleClick(ev.ID)
		}
	}
	return []int32{}, nil
}

func (m *DBusMenu) handleClick(id int32) {
	if id == 1 {
		if m.ctx != nil {
			runtime.WindowShow(m.ctx)
			runtime.WindowUnminimise(m.ctx)
		}
	} else if id == 2 {
		go func() {
			if m.app != nil {
				_ = m.app.Disconnect()
			}
			if m.ctx != nil {
				runtime.Quit(m.ctx)
			}
			os.Exit(0)
		}()
	}
}

func (m *DBusMenu) AboutToShow(id int32) (bool, *dbus.Error) {
	return false, nil
}

func (m *DBusMenu) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	return []int32{}, []int32{}, nil
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
	if err := conn.Export(notifier, "/StatusNotifierItem", "org.kde.StatusNotifierItem"); err != nil {
		fmt.Printf("[NetConnect Tray] Failed to export notifier: %v\n", err)
		return
	}

	menu := &DBusMenu{app: app, ctx: ctx}
	if err := conn.Export(menu, "/MenuBar", "com.canonical.dbusmenu"); err != nil {
		fmt.Printf("[NetConnect Tray] Failed to export dbusmenu: %v\n", err)
	}

	menuPropsSpec := prop.Map{
		"com.canonical.dbusmenu": {
			"Version": {
				Value: uint32(3),
			},
			"TextDirection": {
				Value: "ltr",
			},
			"Status": {
				Value: "normal",
			},
			"IconThemePath": {
				Value: []string{},
			},
		},
	}
	_, _ = prop.Export(conn, "/MenuBar", menuPropsSpec)

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
			"Menu": {
				Value: dbus.ObjectPath("/MenuBar"),
			},
			"ItemIsMenu": {
				Value: true,
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
