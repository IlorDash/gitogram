package tui

import (
	"github.com/IlorDash/gitogram/internal/client"
	"github.com/IlorDash/gitogram/internal/server"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var app = tview.NewApplication()
var text = tview.NewTextView().
	SetTextColor(tcell.ColorGreen).
	SetText("(s) to go to server \n(c) to go to client \n(q) to quit")

var pages = tview.NewPages()

var menuFlex = tview.NewFlex()

var serverForm = tview.NewForm()

func addServerForm() {
	serverForm.AddButton("start server", server.Run)

	serverForm.AddButton("return", func() {
		pages.SwitchToPage("Menu")
	})
}

var clientForm = tview.NewForm()

func addClientForm() {
	var addr string

	clientForm.AddInputField("server address", "", 20, nil, func(newAddr string) {
		addr = newAddr
	})

	clientForm.AddButton("get chats", func() {
		client.GetChats(addr)
	})

	clientForm.AddButton("return", func() {
		pages.SwitchToPage("Menu")
	})
}

func Run() {

	menuFlex.SetDirection(tview.FlexRow).
		AddItem(text, 0, 1, false)

	menuFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		switch event.Rune() {
		case 'q':
			app.Stop()
		case 's':
			serverForm.Clear(true)
			addServerForm()
			pages.SwitchToPage("Server page")
		case 'c':
			clientForm.Clear(true)
			addClientForm()
			pages.SwitchToPage("Client page")
		default:
		}
		return event
	})

	pages.AddPage("Menu", menuFlex, true, true)
	pages.AddPage("Server page", serverForm, true, false)
	pages.AddPage("Client page", clientForm, true, false)

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
