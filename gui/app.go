package main

import (
	"context"
	"fmt"
)

// App struct
type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Connect() {
	fmt.Println("Connecting to NetLink...")
}
