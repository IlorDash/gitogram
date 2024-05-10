package tui

import (
	"log"

	"github.com/IlorDash/gitogram/internal/client"
	"github.com/IlorDash/gitogram/internal/server"
	"github.com/rivo/tview"
)

func createModalForm(form tview.Primitive, height int, width int) tview.Primitive {
	modal := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
	return modal
}

func startServerForm(pages *tview.Pages) func() {
	return func() {
		serverForm := tview.NewForm()
		serverForm.AddButton("Start server", func() {
			ans := server.Run()
			log.Printf("%s", ans)
		})
		serverForm.AddButton("Quit", func() {
			pages.SwitchToPage("main")
			pages.RemovePage("modal")
		})
		serverForm.SetButtonsAlign(tview.AlignCenter)
		serverForm.SetBorder(true).SetTitle("Start server")
		modal := createModalForm(serverForm, 13, 55)
		pages.AddPage("modal", modal, true, true)
	}
}

func connectToServer(pages *tview.Pages) func() {
	return func() {
		var addr string
		connectForm := tview.NewForm()
		connectForm.AddInputField("Server address", "", 20, nil, func(newAddr string) {
			addr = newAddr
		})
		connectForm.AddButton("Connect", func() {
			client.Connect(addr)
		})

		connectForm.AddButton("Quit", func() {
			pages.SwitchToPage("main")
			pages.RemovePage("modal")
		})
		connectForm.SetButtonsAlign(tview.AlignCenter)
		connectForm.SetBorder(true).SetTitle("Connect to server")
		modal := createModalForm(connectForm, 13, 55)
		pages.AddPage("modal", modal, true, true)
	}
}

func createCmdList(app *tview.Application, pages *tview.Pages) *tview.List {
	commandList := tview.NewList()
	commandList.SetBorder(true).SetTitle("Commands")
	commandList.ShowSecondaryText(false)
	commandList.AddItem("Start server", "", 's', startServerForm(pages))
	commandList.AddItem("Connect to server", "", 'o', connectToServer(pages))
	commandList.AddItem("Choose chat", "", 'c', func() {
		// git.Chat
	})
	commandList.AddItem("Members", "", 'm', func() {
		// git.ViewMembers
	})
	commandList.AddItem("Quit", "", 'q', func() {
		// Save config here
		app.Stop()
	})
	return commandList
}

func createLogPanel(app *tview.Application) *tview.TextView {
	panel := tview.NewTextView()
	panel.SetBorder(true).SetTitle("Logs")
	panel.SetChangedFunc(func() {
		app.Draw()
	})
	return panel
}

func createChatPanel(app *tview.Application) *tview.Flex {
	chatPanel := tview.NewFlex().SetDirection(tview.FlexRow)

	chatName := tview.NewTextView()
	chatName.SetText("Chat@")
	chatName.SetTextAlign(tview.AlignLeft)

	chatInfo := tview.NewTable()

	chatInfo.SetCellSimple(0, 0, "Messages:")
	chatInfo.GetCell(0, 0).SetAlign(tview.AlignRight)
	msgNumCell := tview.NewTableCell("0")
	chatInfo.SetCell(0, 1, msgNumCell)

	chatInfo.SetCellSimple(1, 0, "Members:")
	chatInfo.GetCell(1, 0).SetAlign(tview.AlignRight)
	membersNumCell := tview.NewTableCell("0")
	chatInfo.SetCell(1, 1, membersNumCell)

	chatInfoPanel := tview.NewFlex().SetDirection(tview.FlexColumn)
	chatInfoPanel.SetBorder(true)
	chatInfoPanel.AddItem(chatName, 0, 1, false)
	chatInfoPanel.AddItem(chatInfo, 0, 1, false)

	chat := tview.NewTextView()
	chat.SetBorder(true)
	chat.SetChangedFunc(func() {
		app.Draw()
	})

	chatPanel.AddItem(chatInfoPanel, 5, 1, true).
		AddItem(chat, 0, 1, false)

	return chatPanel
}

func createAppLayout(commandList tview.Primitive, outputPanel tview.Primitive) *tview.Flex {

	appLayout := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(commandList, 30, 1, true).
		AddItem(outputPanel, 0, 4, false)

	info := tview.NewTextView()
	info.SetBorder(true)
	info.SetText("Gitogram v0.1 - Copyright 2024 Ilya Orazov <ilordash02@gmail.com>")
	info.SetTextAlign(tview.AlignCenter)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(appLayout, 0, 20, true).
		AddItem(info, 3, 1, false)

	return layout
}

func createApp() *tview.Application {
	app := tview.NewApplication()
	pages := tview.NewPages()

	commandList := createCmdList(app, pages)
	logPanel := createLogPanel(app)
	log.SetOutput(logPanel)

	chatPanel := createChatPanel(app)

	mainPageLayout := createAppLayout(commandList, chatPanel)

	pages.AddPage("main", mainPageLayout, true, true)

	app.SetRoot(pages, true)

	return app
}

func Run() {
	app := createApp()

	if err := app.Run(); err != nil {
		panic(err)
	}
}
