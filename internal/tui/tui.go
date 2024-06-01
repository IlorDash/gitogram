package tui

import (
	"errors"
	"fmt"
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

func chatListUpperStr(n string, t string) string {
	return fmt.Sprintf("%s %s", n, t)
}
func chatListBottomStr(a string, m string) string {
	return fmt.Sprintf("%s: %s", a, m)
}

func createChatList(s *appScreen, p *tview.Pages) (*tview.List, error) {
	list, err := client.ChatsList()
	if err != nil {
		return nil, err
	}

	chatList := tview.NewList()
	chatList.SetBorder(true).SetTitle("Chats")
	chatList.AddItem("New chat +", "", 0, connChat(s, p))

	for _, chat := range list {

		chatList.AddItem(chatListUpperStr(chat.Name, chat.MsgTime),
			chatListBottomStr(chat.Author, chat.LastMsg), 0,
			func() { log.Printf("Selected %s chat\n", chat.Name) })
	}

	return chatList, nil
}

type logLayout struct {
	panel  *tview.Flex
	text   *tview.TextView
	button *tview.Button
}

func createLog(app *tview.Application, p *tview.Pages) *logLayout {
	log := &logLayout{}

	log.text = tview.NewTextView()
	log.text.SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	log.button = tview.NewButton("Return").SetSelectedFunc(func() {
		p.SwitchToPage("main")
	})
	log.button.SetBorder(true).SetRect(0, 0, 22, 3)
	spacer := tview.NewBox()
	emptyBox := tview.NewBox()

	buttonRow := tview.NewFlex().
		AddItem(emptyBox, 0, 1, false).
		AddItem(log.button, 22, 1, false).
		AddItem(emptyBox, 0, 1, false)

	log.panel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(log.text, 0, 1, true).
		AddItem(spacer, 0, 1, false).
		AddItem(buttonRow, 3, 1, false)
	log.panel.SetBorder(true).SetTitle("Logs")
	return log
}

type mainLayout struct {
	chatList *tview.List
	chat     *chatLayout
	cmds     *tview.Flex
}

type focusStruct struct {
	panels []tview.Primitive
	curr   int
}

func (f *focusStruct) setPanel(i int) (tview.Primitive, error) {
	if i > len(f.panels) {
		return nil, errors.New("invalid focus panel")
	}
	f.curr = i
	return f.panels[f.curr], nil
}

type appScreen struct {
	app       *tview.Application
	layout    mainLayout
	log       *logLayout
	focus     focusStruct
	showModal bool
}

func createMainPage(l mainLayout) *tview.Flex {
	innerLayout := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(l.chatList, 0, 1, true).
		AddItem(l.chat.panel, 0, 3, false)

	globalLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(innerLayout, 0, 20, true).AddItem(l.cmds, 3, 1, false)

	globalLayout.SetBorder(true).SetTitle("Gitogram v0.1")

	return globalLayout
}

func addNewChatToList(s *appScreen, c client.BriefChatInfo) {
	s.app.QueueUpdateDraw(func() {
		s.layout.chatList.AddItem(chatListUpperStr(c.Name, c.MsgTime),
			chatListBottomStr(c.Author, c.LastMsg), 0,
			func() { log.Printf("Selected %s chat\n", c.Name) })
	})

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

func connChat(s *appScreen, p *tview.Pages) func() {
	return func() {
		var url string
		getChatForm := tview.NewForm()
		getChatForm.AddInputField("Chat address", "", 50, nil, func(newUrl string) {
			url = newUrl
		})
		getChatForm.AddButton("Get", func() {
			go func() {
				name, memberNum, msgNum, chat, err := client.GetChat(url)
				if err != nil {
					return
				}
				addNewChatToList(s, chat)
				s.chatName(name)
				s.memberNum(memberNum)
				s.msgNum(msgNum)
			}()
		})

		getChatForm.AddButton("Quit", func() {
			s.showModal = false
			p.SwitchToPage("main")
			p.RemovePage("modal")
		})
		getChatForm.SetButtonsAlign(tview.AlignCenter)
		getChatForm.SetBorder(true).SetTitle("Get chat")
		modal := createModalForm(getChatForm, 13, 70)
		s.showModal = true
		p.AddPage("modal", modal, true, true)
	}
}

func (s *appScreen) highlightPanel(p tview.Primitive) error {

	s.layout.chatList.SetBorderColor(tcell.ColorWhite)
	s.layout.chat.panel.SetBorderColor(tcell.ColorWhite)

	switch p {
	case s.layout.chatList:
		s.layout.chatList.SetBorderColor(tcell.ColorGreen)
	case s.layout.chat.panel:
		s.layout.chat.panel.SetBorderColor(tcell.ColorGreen)
	default:
		return errors.New("invalid panel border")
	}
	return nil
}

func (s *appScreen) focusNextPanel() error {
	f := (s.focus.curr + 1) % len(s.focus.panels)
	panel, err := s.focus.setPanel(f)
	if err != nil {
		return err
	}

	s.app.SetFocus(panel)
	s.highlightPanel(panel)
	return nil
}

func showMembers() func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey { return event }
}

func switchToLogs(s *appScreen, p *tview.Pages) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		p.SwitchToPage("log")
		s.app.SetFocus(s.log.button)
		return event
	}
}

func quitApp(s *appScreen) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		s.app.Stop()
		return nil
	}
}

func switchPanel(s *appScreen) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		err := s.focusNextPanel()
		if err != nil {
			log.Fatalln(err)
		}
		return nil
	}
}

type cmd struct {
	name string
	f    func(event *tcell.EventKey) *tcell.EventKey
}

var runeCmds map[rune]cmd
var keyCmds map[tcell.Key]cmd

func initCommands(s *appScreen, p *tview.Pages) {
	runeCmds = make(map[rune]cmd)
	runeCmds['m'] = cmd{name: "Members", f: showMembers()}
	runeCmds['l'] = cmd{name: "Logs", f: switchToLogs(s, p)}
	runeCmds['q'] = cmd{name: "Quit", f: quitApp(s)}

	keyCmds = make(map[tcell.Key]cmd)
	keyCmds[tcell.KeyTab] = cmd{name: "", f: switchPanel(s)}
}

func setKeyboardHandler(s *appScreen, p *tview.Pages) {
	s.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currPage, _ := p.GetFrontPage()
		if s.showModal || currPage != "main" {
			return event
		}

		cmd, ok := runeCmds[event.Rune()]
		if ok {
			return cmd.f(event)
		}
		cmd, ok = keyCmds[event.Key()]
		if ok {
			return cmd.f(event)
		}
		return event
	})
}

func createCommands(s *appScreen, p *tview.Pages) *tview.Flex {

	initCommands(s, p)

	cmdContainer := tview.NewFlex()
	for r, cmd := range runeCmds {
		list := tview.NewList().
			AddItem(cmd.name, "", r, nil)
		cmdContainer.AddItem(list, 0, 1, true)
	}
	cmdContainer.SetDirection(tview.FlexColumn).SetBorder(true)

	setKeyboardHandler(s, p)

	return cmdContainer
}

var dialogue *log.Logger

func setOutputs(s *appScreen) {
	dialogue = log.New(s.layout.chat.dialogue, "", log.LstdFlags)
	dialogue.Println("You got mail!")
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lshortfile)
	log.SetOutput(s.log.text)
	log.Println("You got log")
}

func createApp() (*tview.Application, error) {
	screen := &appScreen{}
	screen.app = tview.NewApplication()
	pages := tview.NewPages()

	var err error

	screen.layout.chatList, err = createChatList(screen, pages)
	if err != nil {
		return nil, err
	}
	screen.layout.chat = createChatLayout(screen.app)
	screen.layout.cmds = createCommands(screen, pages)
	screen.showModal = false

	screen.focus.panels = []tview.Primitive{screen.layout.chatList, screen.layout.chat.panel}
	mainPage := createMainPage(screen.layout)
	screen.log = createLog(screen.app, pages)
	screen.highlightPanel(screen.layout.chatList)
	setOutputs(screen)

	pages.AddPage("main", mainPage, true, true)
	pages.AddPage("log", screen.log.panel, true, false)

	screen.app.SetRoot(pages, true)

	return screen.app, nil
}

func Run() {
	app, err := createApp()

	if err != nil {
		panic(err)
	}

	if err := app.Run(); err != nil {
		panic(err)
	}
}
