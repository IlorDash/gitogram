package tui

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/IlorDash/gitogram/internal/appConfig"
	"github.com/IlorDash/gitogram/internal/client"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type chatInfo struct {
	table      *tview.Table
	msgNum     *tview.TableCell
	membersNum *tview.TableCell
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
	message  *tview.InputField
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
	h.info.membersNum = tview.NewTableCell("0")
	h.info.table.SetCell(1, 1, h.info.membersNum)

	h.panel = tview.NewFlex().SetDirection(tview.FlexColumn)
	h.panel.SetBorder(true)
	h.panel.AddItem(h.name, 0, 1, false)
	h.panel.AddItem(h.info.table, 0, 1, false)

	return h
}

func createChatLayout(s *appScreen) *chatLayout {
	c := &chatLayout{}
	c.panel = tview.NewFlex().SetDirection(tview.FlexRow)
	c.panel.SetBorder(true)

	c.header = *createChatHeader()

	c.dialogue = tview.NewTextView()
	c.dialogue.SetChangedFunc(func() {
		s.app.Draw()
	})
	c.dialogue.SetDynamicColors(true).SetBorder(true)

	var msg string
	c.message = tview.NewInputField().
		SetPlaceholder("Write a message...").
		SetFieldTextColor(tcell.ColorSilver).
		SetPlaceholderTextColor(tcell.ColorGray).
		SetChangedFunc(func(newMsg string) {
			msg = newMsg
		}).
		SetDoneFunc(func(key tcell.Key) {
			msgInfo, err := client.SendMsg(msg)
			if err != nil {
				appConfig.LogErr(err, "failed to send msg")
				return
			}
			chat, err := client.GetCurrChat()
			if err != nil {
				appConfig.LogErr(err, "failed get current Chat")
				return
			}
			updCurrChatInList(s, chat, msgInfo)
			msgHandler.Print(msgInfo)
		})

	c.panel.AddItem(c.header.panel, 0, 2, false).
		AddItem(c.dialogue, 0, 8, false).
		AddItem(c.message, 0, 1, false)

	return c
}

func chatListUpperStr(n string, t string) string {
	return fmt.Sprintf("%s %s", n, t)
}
func chatListBottomStr(a string, m string) string {
	return fmt.Sprintf("%s: %s", a, m)
}

func handleChatSelected(s *appScreen, c client.Chat) {
	go func() {
		log.Printf("Selected %s chat\n", c.Name)
		client.SelectChat(c)
		s.main.selectChatIndex = s.main.chatList.GetCurrentItem()
		s.chatName(c.Name)
		s.membersNum(c.MembersNum)
		s.msgNum(c.MsgNum)
	}()
}

func chatListRelativeTime(t time.Time) string {
	if time.Since(t) < 24*time.Hour {
		return t.Format("15:04")
	} else if time.Since(t) < 7*24*time.Hour {
		return t.Format("Monday")
	} else {
		return t.Format("02.01.2006")
	}
}

func addNewChatToList(s *appScreen, list *tview.List, chat client.Chat, lastMsg client.Message) {
	list.AddItem(chatListUpperStr(chat.Name, chatListRelativeTime(lastMsg.Time)),
		chatListBottomStr(lastMsg.Author, lastMsg.Text), 0,
		func() { handleChatSelected(s, chat) })
}

func updCurrChatInList(s *appScreen, chat client.Chat, lastMsg client.Message) {
	index := s.main.selectChatIndex
	s.main.chatList.RemoveItem(index)

	s.main.chatList.InsertItem(index, chatListUpperStr(chat.Name, chatListRelativeTime(lastMsg.Time)),
		chatListBottomStr(lastMsg.Author, lastMsg.Text), 0,
		func() { handleChatSelected(s, chat) })
	s.main.chatList.SetCurrentItem(index)
}

func createChatList(s *appScreen, p *tview.Pages) (*tview.List, error) {
	chats, lastMsgs, err := client.CollectChats()
	if err != nil {
		return nil, err
	}

	chatList := tview.NewList()
	chatList.SetBorder(true).SetTitle("Chats")
	chatList.AddItem("New chat +", "", 0, addChat(s, p))

	for i := 0; i < len(chats) && i < len(lastMsgs); i++ {
		index := i
		addNewChatToList(s, chatList, chats[index], lastMsgs[index])
	}

	return chatList, nil
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

type logLayout struct {
	panel  *tview.Flex
	text   *tview.TextView
	button *tview.Button
	focus  *focusStruct
}

func createLog(s *appScreen, p *tview.Pages) *logLayout {
	log := &logLayout{}

	log.text = tview.NewTextView()
	log.text.SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			s.app.Draw()
		})
	log.text.SetBorder(true)

	log.button = tview.NewButton("Return").SetSelectedFunc(func() {
		p.SwitchToPage("main")
		s.currPage, _ = p.GetFrontPage()
	})
	log.button.SetBorder(true)

	buttonRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 6, false).
		AddItem(log.button, 0, 2, false).
		AddItem(tview.NewBox(), 0, 6, false)

	log.panel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(log.text, 0, 10, true).
		AddItem(buttonRow, 0, 1, false)
	log.panel.SetBorder(true).SetTitle("Logs")

	return log
}

const msgFocusNum int = 2

type mainLayout struct {
	panel           *tview.Flex
	chatList        *tview.List
	selectChatIndex int
	chat            *chatLayout
	cmds            *tview.Flex
	focus           *focusStruct
}

func createMain(s *appScreen, p *tview.Pages) (*mainLayout, error) {
	var err error
	main := &mainLayout{}

	main.chatList, err = createChatList(s, p)
	if err != nil {
		return nil, err
	}
	main.selectChatIndex = 0
	main.chat = createChatLayout(s)
	main.cmds = createCommands(s, p)

	innerLayout := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(main.chatList, 0, 1, true).
		AddItem(main.chat.panel, 0, 3, false)

	main.panel = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(innerLayout, 0, 20, true).AddItem(main.cmds, 3, 1, false)

	main.panel.SetBorder(true).SetTitle("Gitogram v0.1")

	main.highlightPanel(main.chatList)

	return main, nil
}

type appScreen struct {
	app       *tview.Application
	main      *mainLayout
	log       *logLayout
	showModal bool
	currPage  string
}

func queueUpdateAndDraw(app *tview.Application, f func()) {
	app.QueueUpdateDraw(f)
}

func (s *appScreen) chatName(name string) {
	queueUpdateAndDraw(s.app, func() {
		h := s.main.chat.header
		if h.name != nil {
			h.name.SetText(name)
		}
	})
}

func (s *appScreen) msgNum(num int) {
	queueUpdateAndDraw(s.app, func() {
		h := s.main.chat.header
		if h.info.msgNum != nil {
			h.info.msgNum.SetText(strconv.Itoa(num))
		}
	})
}

func (s *appScreen) membersNum(num int) {
	queueUpdateAndDraw(s.app, func() {
		h := s.main.chat.header
		if h.info.membersNum != nil {
			h.info.membersNum.SetText(strconv.Itoa(num))
		}
	})
}

func createModalForm(form tview.Primitive, height int, width int) tview.Primitive {
	modal := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, height, 10, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
	return modal
}

func addChat(s *appScreen, p *tview.Pages) func() {
	return func() {
		var chatUrl string
		getChatForm := tview.NewForm()
		getChatForm.AddInputField("Chat address", "", 50, nil, func(newUrl string) {
			chatUrl = newUrl
		})
		getChatForm.AddButton("Add", func() {
			go func() {
				chat, lastMsg, err := client.AddChat(chatUrl)
				if err != nil {
					return
				}
				s.app.QueueUpdateDraw(func() {
					addNewChatToList(s, s.main.chatList, chat, lastMsg)
				})
			}()
		})

		getChatForm.AddButton("Quit", func() {
			s.showModal = false
			p.SwitchToPage("main")
			p.RemovePage("modal")
		})
		getChatForm.SetButtonsAlign(tview.AlignCenter)
		getChatForm.SetBorder(true).SetTitle("Add Chat")
		modal := createModalForm(getChatForm, 7, 70)
		s.showModal = true
		p.AddPage("modal", modal, true, true)
	}
}

func (l *logLayout) highlightPanel(p tview.Primitive) error {

	l.text.SetBorderColor(tcell.ColorWhite)
	l.button.SetBorderColor(tcell.ColorWhite)

	switch p {
	case l.text:
		l.text.SetBorderColor(tcell.ColorGreen)
	case l.button:
		l.button.SetBorderColor(tcell.ColorGreen)
	default:
		return errors.New("invalid panel border")
	}
	return nil
}

func (m *mainLayout) highlightPanel(p tview.Primitive) error {

	m.chatList.SetBorderColor(tcell.ColorWhite)
	m.chat.dialogue.SetBorderColor(tcell.ColorWhite)
	m.chat.message.SetPlaceholderTextColor(tcell.ColorGray)

	switch p {
	case m.chatList:
		m.chatList.SetBorderColor(tcell.ColorGreen)
	case m.chat.dialogue:
		m.chat.dialogue.SetBorderColor(tcell.ColorGreen)
	case m.chat.message:
		m.chat.message.SetPlaceholderTextColor(tcell.ColorSilver)
	default:
		return errors.New("invalid panel border")
	}
	return nil
}

func (s *appScreen) focusNextPanel() error {
	if s.currPage == "main" {
		focus := s.main.focus
		f := (focus.curr + 1) % len(focus.panels)
		panel, err := focus.setPanel(f)
		if err != nil {
			return err
		}

		s.app.SetFocus(panel)
		s.main.highlightPanel(panel)
	} else if s.currPage == "log" {
		focus := s.log.focus
		f := (focus.curr + 1) % len(focus.panels)
		panel, err := focus.setPanel(f)
		if err != nil {
			return err
		}

		s.app.SetFocus(panel)
		s.log.highlightPanel(panel)
	} else {
		return errors.New("wromg current page")
	}

	return nil
}

func showMembers() func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey { return event }
}

func switchToLogs(s *appScreen, p *tview.Pages) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		p.SwitchToPage("log")
		s.currPage, _ = p.GetFrontPage()
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

func setKeyboardHandler(s *appScreen) {
	s.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if s.showModal {
			return event
		}

		cmd, ok := keyCmds[event.Key()]
		if ok {
			return cmd.f(event)
		}

		if (s.currPage != "main") || (s.main.focus.curr == msgFocusNum) {
			return event
		}

		cmd, ok = runeCmds[event.Rune()]
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

	setKeyboardHandler(s)

	return cmdContainer
}

type tuiMessageHandler struct{ s *appScreen }

// const msgTopLineColor string = "[Blue]"
// const msgBottomLineColor string = "[White]"

var dialogue *log.Logger
var msgHandler client.MsgHandler

func isDifferentDay(d1, d2 time.Time) bool {
	year1, month1, day1 := d1.Date()
	year2, month2, day2 := d2.Date()

	return year1 != year2 || month1 != month2 || day1 != day2
}

var prevDate time.Time

func newDate(t time.Time) bool {
	if (prevDate == (time.Time{})) || isDifferentDay(prevDate, t) {
		prevDate = t
		return true
	}
	return false
}

func dialogueNewDate(t time.Time) string {
	if time.Now().Year() != t.Year() {
		return t.Format("January 2, 2006")
	}
	return t.Format("January 2")
}

func (h tuiMessageHandler) Print(m client.Message) {
	if newDate(m.Time) {
		dialogue.Println(dialogueNewDate(m.Time) + "\n")
	}

	topLine := m.Author + " " + m.Time.Format("15:04")
	bottomLine := m.Text
	dialogue.Println(topLine + "\n" + bottomLine + "\n")
	h.s.main.chat.dialogue.ScrollToEnd()
}

func setOutputs(s *appScreen) {
	dialogue = log.New(s.main.chat.dialogue, "", 0)
	log.SetFlags(log.LstdFlags)
	log.SetOutput(s.log.text)
	if appConfig.Debug {
		log.Println("You're in Debug mode")
	}

	msgHandler = tuiMessageHandler{s: s}
	client.SetMessageHandler(msgHandler)
}

func createApp() (*tview.Application, error) {
	screen := &appScreen{}
	screen.app = tview.NewApplication()
	pages := tview.NewPages()

	var err error

	screen.showModal = false
	screen.main, err = createMain(screen, pages)
	if err != nil {
		return nil, err
	}
	screen.main.focus = &focusStruct{}
	screen.main.focus.panels = []tview.Primitive{screen.main.chatList, screen.main.chat.dialogue, screen.main.chat.message}

	screen.log = createLog(screen, pages)
	screen.log.focus = &focusStruct{}
	screen.log.focus.panels = []tview.Primitive{screen.log.text, screen.log.button}

	setOutputs(screen)

	pages.AddPage("main", screen.main.panel, true, true)
	pages.AddPage("log", screen.log.panel, true, false)
	screen.currPage, _ = pages.GetFrontPage()

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
