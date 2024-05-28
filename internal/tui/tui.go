package tui

import (
	"log"

	"github.com/IlorDash/gitogram/internal/client"
	"github.com/rivo/tview"
)

func queueUpdateAndDraw(app *tview.Application, f func()) {
	app.QueueUpdateDraw(f)
}

func (ui *chatUI) chatName(s string) {
	queueUpdateAndDraw(ui.app, func() {
		if ui.header.name != nil {
			ui.header.name.SetText(s)
		}
	})
}

func (ui *chatUI) msgNum(s string) {
	queueUpdateAndDraw(ui.app, func() {
		if ui.header.msgNumCell != nil {
			ui.header.msgNumCell.SetText(s)
		}
	})
}

func (ui *chatUI) memberNum(s string) {
	queueUpdateAndDraw(ui.app, func() {
		if ui.header.memberNumCell != nil {
			ui.header.memberNumCell.SetText(s)
		}
	})
}

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

func getChat(pages *tview.Pages, c *chatUI) func() {
	return func() {
		var url string
		getChatForm := tview.NewForm()
		getChatForm.AddInputField("Chat address", "", 20, nil, func(newUrl string) {
			url = newUrl
		})
		getChatForm.AddButton("Get", func() {
			go func() {
				name, memberNum, msgNum, err := client.GetChat(url)
				if err != nil {
					return
				}
				c.chatName(name)
				c.memberNum(memberNum)
				c.msgNum(msgNum)
			}()
		})

		getChatForm.AddButton("Quit", func() {
			pages.SwitchToPage("main")
			pages.RemovePage("modal")
		})
		getChatForm.SetButtonsAlign(tview.AlignCenter)
		getChatForm.SetBorder(true).SetTitle("Get chat")
		modal := createModalForm(getChatForm, 13, 55)
		pages.AddPage("modal", modal, true, true)
	}
}

type chatHeader struct {
	name          *tview.TextView
	msgNumCell    *tview.TableCell
	memberNumCell *tview.TableCell
}

type chatUI struct {
	app    *tview.Application
	panel  *tview.Flex
	header chatHeader
	chat   *tview.TextView
}

func createChatUI(app *tview.Application) (c *chatUI) {
	chatPanel := tview.NewFlex().SetDirection(tview.FlexRow)

	c = &chatUI{}
	c.app = app
	c.panel = chatPanel

	c.header.name = tview.NewTextView()
	c.header.name.SetText("Chat@")
	c.header.name.SetTextAlign(tview.AlignLeft)

	chatInfo := tview.NewTable()

	chatInfo.SetCellSimple(0, 0, "Messages:")
	chatInfo.GetCell(0, 0).SetAlign(tview.AlignRight)
	c.header.msgNumCell = tview.NewTableCell("0")
	chatInfo.SetCell(0, 1, c.header.msgNumCell)

	chatInfo.SetCellSimple(1, 0, "Members:")
	chatInfo.GetCell(1, 0).SetAlign(tview.AlignRight)
	c.header.memberNumCell = tview.NewTableCell("0")
	chatInfo.SetCell(1, 1, c.header.memberNumCell)

	chatInfoPanel := tview.NewFlex().SetDirection(tview.FlexColumn)
	chatInfoPanel.SetBorder(true)
	chatInfoPanel.AddItem(c.header.name, 0, 1, false)
	chatInfoPanel.AddItem(chatInfo, 0, 1, false)

	c.chat = tview.NewTextView()
	c.chat.SetBorder(true)
	c.chat.SetChangedFunc(func() {
		app.Draw()
	})

	chatPanel.AddItem(chatInfoPanel, 5, 1, true).
		AddItem(c.chat, 0, 1, false)

	return c
}

func createCmdList(app *tview.Application, pages *tview.Pages, c *chatUI) *tview.List {
	commandList := tview.NewList()
	commandList.SetBorder(true).SetTitle("Commands")
	commandList.ShowSecondaryText(false)
	commandList.AddItem("Get chat", "", 'g', getChat(pages, c))
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

type logLayout struct {
	panel *tview.Flex
	text  *tview.TextView
}

func createLog(app *tview.Application) *logLayout {

	log := &logLayout{}

	log.text = tview.NewTextView()
	log.text.SetBorder(true).SetTitle("Logs")
	log.text.SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	buttons := tview.NewForm().
		AddButton("Scroll Down", func() {
			log.text.ScrollToEnd()
		}).
		AddButton("Scroll Up", func() {
			log.text.ScrollToBeginning()
		})

	log.panel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(log.text, 0, 1, true).
		AddItem(buttons, 10, 1, false)
	return log
}

type appLayout struct {
	cmdList *tview.List
	logView *logLayout
	chat    *tview.Flex
}

func createAppLayout(l *appLayout) *tview.Flex {

	appLayout := tview.NewFlex().SetDirection(tview.FlexColumn).AddItem(
		tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(l.cmdList, 0, 1, true).
			AddItem(l.logView.panel, 0, 1, true),
		0, 1, true).
		AddItem(l.chat, 0, 3, false)

	footer := tview.NewTextView()
	footer.SetBorder(true)
	footer.SetText("Gitogram v0.1 - Copyright 2024 Ilya Orazov <ilordash02@gmail.com>")
	footer.SetTextAlign(tview.AlignCenter)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(appLayout, 0, 20, true).
		AddItem(footer, 3, 1, false)

	return layout
}

func createApp() *tview.Application {
	app := tview.NewApplication()
	pages := tview.NewPages()

	layout := &appLayout{}

	chatUI := createChatUI(app)

	layout.chat = chatUI.panel
	layout.cmdList = createCmdList(app, pages, chatUI)
	layout.logView = createLog(app)

	msg := log.New(chatUI.chat, "", log.LstdFlags)
	msg.Println("You got mail!")
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lshortfile)
	log.SetOutput(layout.logView.text)

	mainPageLayout := createAppLayout(layout)

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
