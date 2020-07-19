package main

import (
	"net/rpc"
)

// Command responce for registering rpc functions
type Command struct {
	window *Window
}

// AddBlock add block
func (c *Command) AddBlock(a *AddBlock, res *int) (err error) {
	err = c.window.addBlock(*a)
	*res = errToInt(err)
	return
}

// AddCSS add css
func (c *Command) AddCSS(a *AddCSS, res *int) (err error) {
	err = c.window.addCSS(*a)
	*res = errToInt(err)
	return
}

// AddMenu add menu
func (c *Command) AddMenu(a *AddMenu, res *int) (err error) {
	err = c.window.addMenu(*a)
	*res = errToInt(err)
	return
}

// Update update block
func (c *Command) Update(a *Update, res *int) (err error) {
	err = c.window.updateBlock(*a)
	*res = errToInt(err)
	return
}

// Remove remove block
func (c *Command) Remove(a *Remove, res *int) (err error) {
	err = c.window.removeBlock(*a)
	*res = errToInt(err)
	return
}

// RegisterCommandControl creates new command control instance
func RegisterCommandControl(server *rpc.Server, window *Window) (c *Command, err error) {
	c = &Command{window}
	err = server.Register(c)
	return
}

// if err != nil returns -1
func errToInt(err error) int {
	if err != nil {
		return -1
	}
	return 0
}
