package tui

import (
	"errors"
	"log"

	"github.com/IlorDash/gitogram/internal/client"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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
	c = &chatUI{}
	c.app = app
	c.panel = tview.NewFlex().SetDirection(tview.FlexRow)
	c.panel.SetBorder(true)

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
	c.chat.SetChangedFunc(func() {
		app.Draw()
	})
	c.chat.SetBorder(true)

	c.panel.AddItem(chatInfoPanel, 5, 1, false).
		AddItem(c.chat, 0, 1, false)

	return c
}

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

func getChat(s *screen, pages *tview.Pages, c *chatUI) func() {
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
			s.showModal = false
			pages.SwitchToPage("main")
			pages.RemovePage("modal")
		})
		getChatForm.SetButtonsAlign(tview.AlignCenter)
		getChatForm.SetBorder(true).SetTitle("Get chat")
		modal := createModalForm(getChatForm, 13, 55)
		s.showModal = true
		pages.AddPage("modal", modal, true, true)
	}
}

func createCmdList(s *screen, pages *tview.Pages, c *chatUI) *tview.List {
	commandList := tview.NewList()
	commandList.SetBorder(true).SetTitle("Commands")
	commandList.ShowSecondaryText(false)
	commandList.AddItem("Get chat", "", 'g', getChat(s, pages, c))
	commandList.AddItem("Choose chat", "", 'c', func() {
		// git.Chat
	})
	commandList.AddItem("Members", "", 'm', func() {
		// git.ViewMembers
	})
	commandList.AddItem("Quit", "", 'q', func() {
		// Save config here
		s.app.Stop()
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
	log.text.SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	log.panel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(log.text, 0, 1, true)
	log.panel.SetBorder(true).SetTitle("Logs")
	return log
}

type screenLayout struct {
	cmdList *tview.List
	logView *logLayout
	chat    *tview.Flex
}

type screen struct {
	app        *tview.Application
	layout     screenLayout
	panels     []tview.Primitive
	focusPanel tview.Primitive
	showModal  bool
}

func createScreenLayout(l screenLayout) *tview.Flex {
	screenLayout := tview.NewFlex().SetDirection(tview.FlexColumn).AddItem(
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
		AddItem(screenLayout, 0, 20, true).
		AddItem(footer, 3, 1, false)

	return layout
}

func (s *screen) highlightPanel(p tview.Primitive) error {

	s.layout.chat.SetBorderColor(tcell.ColorWhite)
	s.layout.cmdList.SetBorderColor(tcell.ColorWhite)
	s.layout.logView.panel.SetBorderColor(tcell.ColorWhite)

	switch p {
	case s.layout.chat:
		s.layout.chat.SetBorderColor(tcell.ColorGreen)
	case s.layout.cmdList:
		s.layout.cmdList.SetBorderColor(tcell.ColorGreen)
	case s.layout.logView.panel:
		s.layout.logView.panel.SetBorderColor(tcell.ColorGreen)
	default:
		return errors.New("invalid panel border")
	}
	return nil
}

func (s *screen) setFocus(i int) error {
	if i > len(s.panels) {
		return errors.New("invalid screen panel")
	}
	s.app.SetFocus(s.panels[i])
	s.focusPanel = s.panels[i]
	s.highlightPanel(s.panels[i])
	return nil
}

func setKeyboardHandler(s *screen) {
	i := 1

	s.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		switch event.Key() {
		case tcell.KeyTab:
			if s.showModal {
				return event
			}
			i = (i + 1) % len(s.panels)
			err := s.setFocus(i)
			if err != nil {
				log.Fatalln(err)
			}
			return nil
		default:
			return event
		}
	})
}

func createApp() *tview.Application {

	s := &screen{}
	s.app = tview.NewApplication()
	pages := tview.NewPages()

	chatUI := createChatUI(s.app)

	s.layout.chat = chatUI.panel
	s.layout.cmdList = createCmdList(s, pages, chatUI)
	s.layout.logView = createLog(s.app)
	s.showModal = false

	s.panels = []tview.Primitive{s.layout.chat, s.layout.cmdList, s.layout.logView.panel}

	msg := log.New(chatUI.chat, "", log.LstdFlags)
	msg.Println("You got mail!")
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lshortfile)
	log.SetOutput(s.layout.logView.text)

	mainPageLayout := createScreenLayout(s.layout)

	s.highlightPanel(s.layout.cmdList)

	setKeyboardHandler(s)

	pages.AddPage("main", mainPageLayout, true, true)

	s.app.SetRoot(pages, true)

	return s.app
}

func Run() {
	app := createApp()

	if err := app.Run(); err != nil {
		panic(err)
	}
}
