package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/cep21/xdgbasedir"
	"github.com/gotk3/gotk3/gtk"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	socket = "/tmp/vbar-command"
)

var (
	app = kingpin.New("vbar", "A bar.")

	commandStart = app.Command("start", "Start vbar.")

	commandAddCSS   = app.Command("add-css", "Add CSS.")
	flagAddCSSClass = commandAddCSS.Flag("class", "CSS Class name.").Required().String()
	flagAddCSSValue = commandAddCSS.Flag("css", "CSS value.").Required().String()

	commandAddBlock          = app.Command("add-block", "Add a new block.")
	flagAddBlockName         = commandAddBlock.Flag("name", "Block name.").Required().String()
	flagAddBlockLeft         = commandAddBlock.Flag("left", "Add block to the left.").Bool()
	flagAddBlockCenter       = commandAddBlock.Flag("center", "Add block to the center.").Bool()
	flagAddBlockRight        = commandAddBlock.Flag("right", "Add block to the right.").Bool()
	flagAddBlockText         = commandAddBlock.Flag("text", "Block text.").String()
	flagAddBlockCommand      = commandAddBlock.Flag("command", "Command to execute.").String()
	flagAddBlockTailCommand  = commandAddBlock.Flag("tail-command", "Command to tail.").String()
	flagAddBlockInterval     = commandAddBlock.Flag("interval", "Interval in seconds to execute command.").Int()
	flagAddBlockClickCommand = commandAddBlock.Flag("click-command", "Command to execute when clicking on the block.").String()

	commandAddMenu       = app.Command("add-menu", "Add a menu to a block.")
	flagAddMenuBlockName = commandAddMenu.Flag("name", "Block name.").Required().String()
	flagAddMenuText      = commandAddMenu.Flag("text", "Menu text.").Required().String()
	flagAddMenuCommand   = commandAddMenu.Flag("command", "Command to execute when activating the menu.").Required().String()

	commandUpdate       = app.Command("update", "Trigger a block update.")
	flagUpdateBlockName = commandUpdate.Flag("name", "Block name.").Required().String()

	commandRemove       = app.Command("remove", "Remove a block.")
	flagRemoveBlockName = commandRemove.Flag("name", "Block name.").Required().String()

	window *Window
	mutex  = &sync.Mutex{}
)

func main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case commandStart.FullCommand():
		launch()
	case commandAddCSS.FullCommand():
		err := rpcClient("Command.AddCSS", &AddCSS{
			Class: *flagAddCSSClass,
			Value: *flagAddCSSValue,
		})
		if err != nil {
			log.Panicf("add-css err %v", err)
		}
	case commandAddBlock.FullCommand():
		err := rpcClient("Command.AddBlock", &AddBlock{
			Name:         *flagAddBlockName,
			Text:         *flagAddBlockText,
			Left:         *flagAddBlockLeft,
			Center:       *flagAddBlockCenter,
			Right:        *flagAddBlockRight,
			Command:      *flagAddBlockCommand,
			TailCommand:  *flagAddBlockTailCommand,
			Interval:     *flagAddBlockInterval,
			ClickCommand: *flagAddBlockClickCommand,
		})
		if err != nil {
			log.Panicf("add-block err %v", err)
		}
	case commandAddMenu.FullCommand():
		err := rpcClient("Command.AddMenu", &AddMenu{
			Name:    *flagAddMenuBlockName,
			Text:    *flagAddMenuText,
			Command: *flagAddMenuCommand,
		})
		if err != nil {
			log.Panicf("add-menu err %v", err)
		}
	case commandUpdate.FullCommand():
		err := rpcClient("Command.Update", &Update{
			Name: *flagUpdateBlockName,
		})
		if err != nil {
			log.Panicf("update err %v", err)
		}
	case commandRemove.FullCommand():
		err := rpcClient("Command.Remove", &Remove{
			Name: *flagRemoveBlockName,
		})
		if err != nil {
			log.Panicf("remove err %v", err)
		}
	}
}

func launch() {
	gtk.Init(nil)

	w, err := WindowNew()
	if err != nil {
		log.Panic(err)
	}
	window = w

	control := make(chan int)
	// create command listener
	go func() {
		server := rpc.NewServer()
		_, errC := RegisterCommandControl(server, window)
		if errC != nil {
			log.Panicf("can't register rpc commands %v", errC)
		}
		// remove old socket if it exists NOTE! old vbar instance need to close manualy
		os.Remove(socket)
		listen, errL := net.Listen("unix", socket)
		if errL != nil {
			log.Panicf("can't create command listener %v", errL)
		}
		defer listen.Close()
		// satrt accepting on separate gorotine
		go server.Accept(listen)
		// and wait control signal to stop accepting
		<-control
	}()

	go func() {
		err = executeConfig()
		if err != nil {
			log.Panic(err)
		}
	}()

	// add signal handler for proper close
	go func() {
		c := make(chan os.Signal, 1)

		signal.Notify(c, os.Interrupt, syscall.SIGABRT)
		<-c
		// stops all gorotines wating control
		close(control)
		// close qtk app
		go gtk.MainQuit()
	}()

	gtk.Main()
}

func rpcClient(command string, args interface{}) (err error) {
	conn, errC := net.Dial("unix", socket)
	if errC != nil {
		log.Panicf("can't create command client %v", errC)
	}

	client := rpc.NewClient(conn)
	defer client.Close()

	var res int
	err = client.Call(command, args, &res)
	if res != 0 {
		err = fmt.Errorf("can't do command - error")
	}

	return
}

func executeConfig() error {
	configurationDirectory, err := xdgbasedir.ConfigHomeDirectory()
	if err != nil {
		return err
	}
	configurationFilePath := path.Join(configurationDirectory, "vbar", "vbarrc")

	cmd := exec.Command("/bin/bash", "-c", configurationFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
