package tui

import (
	"errors"
	"log"
	"strconv"

	"github.com/IlorDash/gitogram/internal/client"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type chatInfo struct {
	table     *tview.Table
	msgNum    *tview.TableCell
	memberNum *tview.TableCell
}

type chatHeader struct {
	panel *tview.Flex
	name  *tview.TextView
	info  chatInfo
}

type chatLayout struct {
	panel    *tview.Flex
	header   chatHeader
	dialogue *tview.TextView
}

func createChatHeader() *chatHeader {
	h := &chatHeader{}

	h.name = tview.NewTextView()
	h.name.SetText("Chat@")
	h.name.SetTextAlign(tview.AlignLeft)

	h.info.table = tview.NewTable()

	h.info.table.SetCellSimple(0, 0, "Messages:")
	h.info.table.GetCell(0, 0).SetAlign(tview.AlignRight)
	h.info.msgNum = tview.NewTableCell("0")
	h.info.table.SetCell(0, 1, h.info.msgNum)

	h.info.table.SetCellSimple(1, 0, "Members:")
	h.info.table.GetCell(1, 0).SetAlign(tview.AlignRight)
	h.info.memberNum = tview.NewTableCell("0")
	h.info.table.SetCell(1, 1, h.info.memberNum)

	h.panel = tview.NewFlex().SetDirection(tview.FlexColumn)
	h.panel.SetBorder(true)
	h.panel.AddItem(h.name, 0, 1, false)
	h.panel.AddItem(h.info.table, 0, 1, false)

	return h
}

func createChatLayout(app *tview.Application) *chatLayout {
	c := &chatLayout{}
	c.panel = tview.NewFlex().SetDirection(tview.FlexRow)
	c.panel.SetBorder(true)

	c.header = *createChatHeader()

	c.dialogue = tview.NewTextView()
	c.dialogue.SetChangedFunc(func() {
		app.Draw()
	})
	c.dialogue.SetBorder(true)

	c.panel.AddItem(c.header.panel, 5, 1, false).
		AddItem(c.dialogue, 0, 1, false)

	return c
}

func createCmdList(s *appScreen, pages *tview.Pages) *tview.List {
	commandList := tview.NewList()
	commandList.SetBorder(true).SetTitle("Commands")
	commandList.ShowSecondaryText(false)
	commandList.AddItem("Get chat", "", 'g', getChat(s, pages))
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

type mainLayout struct {
	logView *logLayout
	chat    *chatLayout
	cmds    *tview.List
}

type appScreen struct {
	app        *tview.Application
	layout     mainLayout
	panels     []tview.Primitive
	focusPanel tview.Primitive
	showModal  bool
}

func createMainPage(l mainLayout) *tview.Flex {
	innerLayout := tview.NewFlex().SetDirection(tview.FlexColumn).AddItem(
		tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(l.cmds, 0, 1, true).
			AddItem(l.logView.panel, 0, 1, true),
		0, 1, true).
		AddItem(l.chat.panel, 0, 3, false)

	footer := tview.NewTextView()
	footer.SetBorder(true)
	footer.SetText("Gitogram v0.1 - Copyright 2024 Ilya Orazov <ilordash02@gmail.com>")
	footer.SetTextAlign(tview.AlignCenter)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(innerLayout, 0, 20, true).
		AddItem(footer, 3, 1, false)

	return layout
}

func queueUpdateAndDraw(app *tview.Application, f func()) {
	app.QueueUpdateDraw(f)
}

func (s *appScreen) chatName(name string) {
	queueUpdateAndDraw(s.app, func() {
		h := s.layout.chat.header
		if h.name != nil {
			h.name.SetText(name)
		}
	})
}

func (s *appScreen) msgNum(num int) {
	queueUpdateAndDraw(s.app, func() {
		h := s.layout.chat.header
		if h.info.msgNum != nil {
			h.info.msgNum.SetText(strconv.Itoa(num))
		}
	})
}

func (s *appScreen) memberNum(num int) {
	queueUpdateAndDraw(s.app, func() {
		h := s.layout.chat.header
		if h.info.memberNum != nil {
			h.info.memberNum.SetText(strconv.Itoa(num))
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

func getChat(s *appScreen, pages *tview.Pages) func() {
	return func() {
		var url string
		getChatForm := tview.NewForm()
		getChatForm.AddInputField("Chat address", "", 50, nil, func(newUrl string) {
			url = newUrl
		})
		getChatForm.AddButton("Get", func() {
			go func() {
				name, memberNum, msgNum, err := client.GetChat(url)
				if err != nil {
					return
				}
				s.chatName(name)
				s.memberNum(memberNum)
				s.msgNum(msgNum)
			}()
		})

		getChatForm.AddButton("Quit", func() {
			s.showModal = false
			pages.SwitchToPage("main")
			pages.RemovePage("modal")
		})
		getChatForm.SetButtonsAlign(tview.AlignCenter)
		getChatForm.SetBorder(true).SetTitle("Get chat")
		modal := createModalForm(getChatForm, 13, 70)
		s.showModal = true
		pages.AddPage("modal", modal, true, true)
	}
}

func (s *appScreen) highlightPanel(p tview.Primitive) error {

	s.layout.chat.panel.SetBorderColor(tcell.ColorWhite)
	s.layout.cmds.SetBorderColor(tcell.ColorWhite)
	s.layout.logView.panel.SetBorderColor(tcell.ColorWhite)

	switch p {
	case s.layout.chat.panel:
		s.layout.chat.panel.SetBorderColor(tcell.ColorGreen)
	case s.layout.cmds:
		s.layout.cmds.SetBorderColor(tcell.ColorGreen)
	case s.layout.logView.panel:
		s.layout.logView.panel.SetBorderColor(tcell.ColorGreen)
	default:
		return errors.New("invalid panel border")
	}
	return nil
}

func (s *appScreen) setFocus(i int) error {
	if i > len(s.panels) {
		return errors.New("invalid screen panel")
	}
	s.app.SetFocus(s.panels[i])
	s.focusPanel = s.panels[i]
	s.highlightPanel(s.panels[i])
	return nil
}

func setKeyboardHandler(s *appScreen) {
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

var dialogue *log.Logger

func setOutputs(s *appScreen) {
	dialogue = log.New(s.layout.chat.dialogue, "", log.LstdFlags)
	dialogue.Println("You got mail!")
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lshortfile)
	log.SetOutput(s.layout.logView.text)
	log.Println("You got log")
}

func createApp() *tview.Application {

	screen := &appScreen{}
	screen.app = tview.NewApplication()
	pages := tview.NewPages()

	screen.layout.chat = createChatLayout(screen.app)
	screen.layout.cmds = createCmdList(screen, pages)
	screen.layout.logView = createLog(screen.app)
	screen.showModal = false

	screen.panels = []tview.Primitive{screen.layout.chat.panel, screen.layout.cmds, screen.layout.logView.panel}

	setOutputs(screen)
	mainPage := createMainPage(screen.layout)
	screen.highlightPanel(screen.layout.cmds)
	setKeyboardHandler(screen)
	pages.AddPage("main", mainPage, true, true)

	screen.app.SetRoot(pages, true)

	return screen.app
}

func Run() {
	app := createApp()

	if err := app.Run(); err != nil {
		panic(err)
	}
}
