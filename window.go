package main

import "C"

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

// Window is the container for the bar
type Window struct {
	gtkWindow       *gtk.Window
	gtkBar          *gtk.Grid
	lastLeftBlock   *gtk.EventBox
	lastCenterBlock *gtk.EventBox
	lastRightBlock  *gtk.EventBox
	blocks          []*Block
	cssApplier      *CSSApplier
}

// WindowNew creates a new Window
func WindowNew() (*Window, error) {
	var window = &Window{}

	gtkWindow, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return nil, err
	}
	window.gtkWindow = gtkWindow

	window.gtkWindow.SetAppPaintable(true)
	window.gtkWindow.SetDecorated(false)
	window.gtkWindow.SetResizable(false)
	window.gtkWindow.SetSkipPagerHint(true)
	window.gtkWindow.SetSkipTaskbarHint(true)
	window.gtkWindow.SetTypeHint(gdk.WINDOW_TYPE_HINT_DOCK)
	window.gtkWindow.SetVExpand(false)
	window.gtkWindow.SetPosition(gtk.WIN_POS_NONE)
	window.gtkWindow.Move(0, 0)
	window.gtkWindow.SetSizeRequest(-1, -1)

	window.gtkWindow.Connect("destroy", func() {
		gtk.MainQuit()
	})

	window.gtkWindow.Connect("realize", func() {
		window.gtkWindow.ShowAll()
		updateDimensions(window.gtkWindow, &window.gtkBar.Widget)
	})

	gtkBar, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}
	window.gtkBar = gtkBar

	window.gtkWindow.Add(window.gtkBar)

	applyClass(&window.gtkBar.Widget, "bar")

	enableTransparency(window.gtkWindow)

	return window, nil
}

func (w *Window) addBlock(addBlock AddBlock) error {
	block := &Block{AddBlock: addBlock}

	w.blocks = append(w.blocks, block)

	eventBox, err := gtk.EventBoxNew()
	if err != nil {
		return err
	}
	block.EventBox = eventBox

	label, err := gtk.LabelNew(block.Text)
	if err != nil {
		return err
	}
	applyClass(&label.Widget, "block")
	applyClass(&label.Widget, block.Name)
	block.Label = label
	eventBox.Add(label)

	if block.Left {
		w.addBlockLeft(block)
	} else if block.Center {
		w.addBlockCenter(block)
	} else if block.Right {
		w.addBlockRight(block)
	}

	if block.Command != "" {
		go func() {
			block.updateLabel()

			if block.Interval != 0 {
				duration, _ := time.ParseDuration(fmt.Sprintf("%ds", block.Interval))
				tick := time.Tick(duration)
				go func() {
					for range tick {
						block.updateLabel()
					}
				}()
			}
		}()
	} else if block.TailCommand != "" {
		block.updateLabelForever()
	}

	if block.ClickCommand != "" {
		block.EventBox.Connect("button-release-event", func() {
			go func() {
				cmd := exec.Command("/bin/bash", "-c", block.ClickCommand)
				err := cmd.Run()
				if err != nil {
					log.Printf("ClickCommand finished with error: %v", err)
				}
			}()
		})
	}

	window.gtkWindow.ShowAll()

	return nil
}

func (w *Window) addCSS(addCSS AddCSS) error {
	if w.cssApplier == nil {
		w.cssApplier = &CSSApplier{}
	}

	screen, err := window.gtkWindow.GetScreen()
	if err != nil {
		return err
	}

	err = w.cssApplier.Apply(screen, addCSS)
	if err != nil {
		return err
	}

	return nil
}

func (w *Window) addMenu(addMenu AddMenu) error {
	block := w.findBlock(addMenu.Name)
	if block == nil {
		return errors.New(fmt.Sprintf("Couldn't find block %s.", addMenu.Name))
	}

	if block.Menu == nil {
		menu, err := gtk.MenuNew()
		if err != nil {
			log.Fatal(err)
		}
		block.Menu = menu

		applyClass(&block.Menu.Widget, "menu")

		block.EventBox.Connect("button-release-event", func() {
			popupMenuAt(&block.EventBox.Widget, block.Menu)
		})
	}

	menuItem, err := gtk.MenuItemNewWithLabel(addMenu.Text)
	if err != nil {
		log.Fatal(err)
	}
	menuItem.Connect("activate", func() {
		cmd := exec.Command("/bin/bash", "-c", addMenu.Command)
		err = cmd.Run()
		if err != nil {
			log.Printf("Command finished with error: %v", err)
		}
	})
	block.Menu.Add(menuItem)
	block.Menu.ShowAll()

	return nil
}

func (w *Window) updateBlock(update Update) error {
	block := w.findBlock(update.Name)
	if block == nil {
		return errors.New(fmt.Sprintf("Couldn't find block %s.", update.Name))
	}

	block.updateLabel()
	return nil
}

func (w *Window) findBlock(name string) *Block {
	for _, block := range w.blocks {
		if block.Name == name {
			return block
		}
	}
	return nil
}

func (w *Window) addBlockLeft(block *Block) {
	block.EventBox.SetHAlign(gtk.ALIGN_START)

	if w.lastLeftBlock != nil {
		w.gtkBar.AttachNextTo(block.EventBox, w.lastLeftBlock, gtk.POS_RIGHT, 1, 1)
	} else if w.lastCenterBlock != nil {
		w.gtkBar.AttachNextTo(block.EventBox, w.lastCenterBlock, gtk.POS_LEFT, 1, 1)
	} else if w.lastRightBlock != nil {
		w.gtkBar.AttachNextTo(block.EventBox, w.lastRightBlock, gtk.POS_LEFT, 1, 1)
	} else {
		w.gtkBar.Attach(block.EventBox, 0, 0, 1, 1)
	}
	w.lastLeftBlock = block.EventBox
}

func (w *Window) addBlockCenter(block *Block) {
	block.EventBox.SetHAlign(gtk.ALIGN_CENTER)
	block.EventBox.SetHExpand(true)
	block.Label.SetEllipsize(pango.ELLIPSIZE_END)

	if w.lastCenterBlock != nil {
		w.gtkBar.AttachNextTo(block.EventBox, w.lastCenterBlock, gtk.POS_RIGHT, 1, 1)
	} else if w.lastLeftBlock != nil {
		w.gtkBar.AttachNextTo(block.EventBox, w.lastLeftBlock, gtk.POS_RIGHT, 1, 1)
	} else if w.lastRightBlock != nil {
		w.gtkBar.AttachNextTo(block.EventBox, w.lastRightBlock, gtk.POS_LEFT, 1, 1)
	} else {
		w.gtkBar.Attach(block.EventBox, 0, 0, 1, 1)
	}
	w.lastCenterBlock = block.EventBox

}

func (w *Window) addBlockRight(block *Block) {
	block.EventBox.SetHAlign(gtk.ALIGN_END)

	if w.lastRightBlock != nil {
		w.gtkBar.AttachNextTo(block.EventBox, w.lastRightBlock, gtk.POS_RIGHT, 1, 1)
	} else if w.lastCenterBlock != nil {
		w.gtkBar.AttachNextTo(block.EventBox, w.lastCenterBlock, gtk.POS_RIGHT, 1, 1)
	} else if w.lastLeftBlock != nil {
		w.gtkBar.AttachNextTo(block.EventBox, w.lastLeftBlock, gtk.POS_RIGHT, 1, 1)
	} else {
		w.gtkBar.Attach(block.EventBox, 0, 0, 1, 1)
	}
	w.lastRightBlock = block.EventBox
}
