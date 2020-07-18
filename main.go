package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/exec"
	"path"
	"sync"

	"github.com/cep21/xdgbasedir"
	"github.com/gotk3/gotk3/gtk"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app = kingpin.New("vbar", "A bar.")

	port = app.Flag("port", "Port to use for the command server.").Default("5643").OverrideDefaultFromEnvar("PORT").Int()

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

const (
	socket = "/tmp/vbar_command"
)

func main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case commandStart.FullCommand():
		launch()
	case commandAddCSS.FullCommand():
		var res int
		err := rpcClient().Call("Command.AddCSS", &AddCSS{
			Class: *flagAddCSSClass,
			Value: *flagAddCSSValue,
		}, &res)
		if err != nil || res < 0 {
			log.Panicf("add-css err %v res %d", err, res)
		}
	case commandAddBlock.FullCommand():
		var res int
		err := rpcClient().Call("Command.AddBlock", &AddBlock{
			Name:         *flagAddBlockName,
			Text:         *flagAddBlockText,
			Left:         *flagAddBlockLeft,
			Center:       *flagAddBlockCenter,
			Right:        *flagAddBlockRight,
			Command:      *flagAddBlockCommand,
			TailCommand:  *flagAddBlockTailCommand,
			Interval:     *flagAddBlockInterval,
			ClickCommand: *flagAddBlockClickCommand,
		}, &res)
		if err != nil || res < 0 {
			log.Panicf("add-block err %v res %d", err, res)
		}
	case commandAddMenu.FullCommand():
		var res int
		err := rpcClient().Call("Command.AddMenu", &AddMenu{
			Name:    *flagAddMenuBlockName,
			Text:    *flagAddMenuText,
			Command: *flagAddMenuCommand,
		}, &res)
		if err != nil || res < 0 {
			log.Panicf("add-menu err %v res %d", err, res)
		}
	case commandUpdate.FullCommand():
		var res int
		err := rpcClient().Call("Command.Update", &Update{
			Name: *flagUpdateBlockName,
		}, &res)
		if err != nil || res < 0 {
			log.Panicf("update err %v res %d", err, res)
		}
	case commandRemove.FullCommand():
		var res int
		err := rpcClient().Call("Command.Remove", &Remove{
			Name: *flagRemoveBlockName,
		}, &res)
		if err != nil || res < 0 {
			log.Panicf("remove err %v res %d", err, res)
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

	// create command listener
	go func() {
		server := rpc.NewServer()
		_, errC := RegisterCommandControl(server, window)
		if errC != nil {
			log.Panicf("can't register rpc commands %v", errC)
		}
		listener, errL := net.Listen("unix", socket)
		if errL != nil {
			log.Panicf("can't create command listener %v", errL)
		}
		defer listener.Close()
		server.Accept(listener)
	}()

	go func() {
		err = executeConfig()
		if err != nil {
			log.Panic(err)
		}
	}()

	gtk.Main()
}

func rpcClient() (client *rpc.Client) {
	conn, errC := net.Dial("unix", socket)
	if errC != nil {
		log.Panicf("can't create command client %v", errC)
	}
	client = rpc.NewClient(conn)
	return
}

func sendCommand(path string, command interface{}) {
	jsonValue, err := json.Marshal(command)
	if err != nil {
		log.Panic(err)
	}

	resp, err := http.Post(
		fmt.Sprintf("http://localhost:%d/%s", *port, path),
		"application/json",
		bytes.NewBuffer(jsonValue),
	)
	if err != nil {
		log.Panic(err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var serverResponse ServerResponse
	err = decoder.Decode(&serverResponse)
	if err != nil {
		log.Fatal(err)
	}
	if serverResponse.Error != "" {
		log.Fatal(errors.New(serverResponse.Error))
	}
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
