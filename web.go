package vl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
)

var WebAddr = ":8080"

type sseConn struct {
	ch   chan []byte
	done chan struct{}
}

type eventMsg struct {
	Type    string `json:"type"`
	ID      string `json:"id,omitempty"`
	Action  string `json:"action,omitempty"`
	Checked bool   `json:"checked,omitempty"`
	Index   int    `json:"index,omitempty"`
	Text    string `json:"text,omitempty"`
	Col     int    `json:"col,omitempty"`
	Row     int    `json:"row,omitempty"`
	Key     string `json:"key,omitempty"`
	Ctrl    bool   `json:"ctrl,omitempty"`
	Shift   bool   `json:"shift,omitempty"`
	Alt     bool   `json:"alt,omitempty"`
	Width   int    `json:"width,omitempty"`
	Delta   int    `json:"delta,omitempty"`
}

type sseOutMsg struct {
	Type string `json:"type"`
	HTML string `json:"html"`
}

type WebServer struct {
	mu        sync.Mutex
	root      Widget
	action    chan func()
	chQuit    <-chan struct{}
	srv       *http.Server
	sse       *sseConn
	width     uint
	widgetMap map[string]Widget
	quitKeys  []tcell.Key
	quit      chan struct{}
	quitOnce  sync.Once
	lastHTML  string
}

func (ws *WebServer) shutdown() {
	ws.quitOnce.Do(func() {
		close(ws.quit)
	})
}

func styleHex(s tcell.Style) (fg, bg string) {
	fgC, bgC, _ := s.Decompose()
	return tcellColorHex(fgC), tcellColorHex(bgC)
}

func tcellColorHex(c tcell.Color) string {
	switch c {
	case tcell.ColorBlack:
		return "#000000"
	case tcell.ColorWhite:
		return "#FFFFFF"
	case tcell.ColorRed:
		return "#FF0000"
	case tcell.ColorGreen:
		return "#00FF00"
	case tcell.ColorYellow:
		return "#FFFF00"
	case tcell.ColorBlue:
		return "#0000FF"
	case tcell.ColorDeepPink:
		return "#FF1493"
	}
	return "#FFFFFF"
}

type htmlCtx struct {
	w                    uint
	wm                   map[string]Widget
	btnC, chkC           int
	inpC, rgC, tabC, chC, vcC, menuC, comboE, unknownC int
}

func esc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func widgetToHTML(w Widget, ctx *htmlCtx) string {
	if w == nil {
		return ""
	}
	switch v := w.(type) {
	case *Screen:
		return widgetToHTML(v.root, ctx)

	case *Frame:
		bgFg, bgBg := styleHex(TextStyle)
		var headerHTML string
		if v.Header != nil {
			headerHTML = fmt.Sprintf("<legend style=\"color:%s;background:%s;padding:0 4px;font:inherit;width:100%%;display:block;box-sizing:border-box\">%s</legend>",
				bgFg, bgBg, widgetToHTML(v.Header, ctx))
		}
		content := widgetToHTML(v.root, ctx)
		if v.NoBorder {
			return content
		}
		return fmt.Sprintf("<fieldset style=\"border:1px solid #666;margin:0;padding:2px 4px;color:%s;background:%s;overflow:auto;min-width:0\">%s<div style=\"min-height:1.2em;overflow:auto\">%s</div></fieldset>",
			bgFg, bgBg, headerHTML, content)

	case *List:
		var parts []string
		for _, n := range v.nodes {
			if n.w == nil {
				continue
			}
			parts = append(parts, widgetToHTML(n.w, ctx))
		}
		return "<div style=\"display:flex;flex-direction:column\">" + strings.Join(parts, "") + "</div>"

	case *ListH:
		var parts []string
		for _, n := range v.nodes {
			if n.w == nil {
				continue
			}
			content := widgetToHTML(n.w, ctx)
			parts = append(parts, fmt.Sprintf("<div style=\"flex:1;min-width:0;overflow:hidden\">%s</div>", content))
		}
		return fmt.Sprintf("<div style=\"display:flex;flex-direction:row;flex-wrap:nowrap;gap:2px\">%s</div>", strings.Join(parts, ""))

	case *Scroll:
		if v.root == nil {
			return ""
		}
		maxH := ""
		if v.addlimit && v.hmax > 0 {
			maxH = fmt.Sprintf("max-height:%.0fpx;overflow-y:auto", float64(v.hmax)*16.8)
		}
		inner := widgetToHTML(v.root, ctx)
		return fmt.Sprintf("<div style=\"%s\">%s</div>", maxH, inner)

	case *Button:
		id := fmt.Sprintf("b_%d", ctx.btnC)
		ctx.btnC++
		ctx.wm[id] = v
		text := esc(v.GetText())
		text = strings.ReplaceAll(text, "\n", "<br>")
		return fmt.Sprintf("<button id=\"%s\" style=\"cursor:pointer;padding:0 6px\" onclick=\"fetch('/event',{method:'POST',body:JSON.stringify({type:'widget',id:'%s',action:'click'})})\">%s</button>",
			id, id, text)

	case *CheckBox:
		id := fmt.Sprintf("c_%d", ctx.chkC)
		ctx.chkC++
		ctx.wm[id] = v
		checked := ""
		if v.Checked {
			checked = " checked"
		}
		disabled := ""
		if v.ReadOnly {
			disabled = " disabled"
		}
		text := esc(v.GetText())
		text = strings.ReplaceAll(text, "\n", "<br>")
		return fmt.Sprintf("<label style=\"display:inline-flex;align-items:center;cursor:pointer\"><input type=\"checkbox\" id=\"%s\"%s%s style=\"margin:0 4px 0 0\" onchange=\"fetch('/event',{method:'POST',body:JSON.stringify({type:'widget',id:'%s',action:'change',checked:this.checked})})\"><span>%s</span></label>",
			id, checked, disabled, id, text)

	case *RadioGroup:
		rgID := fmt.Sprintf("rg_%d", ctx.rgC)
		ctx.rgC++
		ctx.wm[rgID+"_"+rgID] = v
		pos := int(v.GetPos())
		if pos >= len(v.list.nodes) {
			pos = 0
		}
		var items []string
		for i, n := range v.list.nodes {
			r, ok := n.w.(*radio)
			if !ok || r == nil {
				continue
			}
			chk := ""
			if i == pos {
				chk = " checked"
			}
			labelText := radioLabel(r)
			if isSimpleText(r.root) {
				items = append(items, fmt.Sprintf(
					`<label style="display:flex;align-items:center;cursor:pointer"><input type="radio" name="%s"%s style="margin:0 4px 0 0" onchange="fetch('/event',{method:'POST',body:JSON.stringify({type:'widget',id:'%s_%s',action:'radio',index:%d})})"><span>%s</span></label>`,
					rgID, chk, rgID, rgID, i, esc(labelText)))
				continue
			}
			contentHTML := widgetToHTML(r.root, ctx)
			disp := "none"
			if i == pos {
				disp = "block"
			}
			items = append(items, fmt.Sprintf(
				`<div style="display:flex;flex-direction:column"><label style="display:flex;align-items:center;cursor:pointer"><input type="radio" name="%s"%s style="margin:0 4px 0 0" onchange="fetch('/event',{method:'POST',body:JSON.stringify({type:'widget',id:'%s_%s',action:'radio',index:%d})})"><span>%s</span></label><div style="display:%s;margin-left:20px">%s</div></div>`,
				rgID, chk, rgID, rgID, i, esc(labelText), disp, contentHTML))
		}
		return fmt.Sprintf("<div style=\"display:flex;flex-direction:column;gap:2px\">%s</div>", strings.Join(items, ""))

	case *InputBox:
		id := fmt.Sprintf("i_%d", ctx.inpC)
		ctx.inpC++
		ctx.wm[id] = v
		text := esc(v.GetText())
		return fmt.Sprintf("<textarea id=\"%s\" style=\"border:1px solid #888;outline:none;resize:none;width:100%%;padding:0 4px\" oninput=\"fetch('/event',{method:'POST',body:JSON.stringify({type:'widget',id:'%s',action:'input',text:this.value})})\">%s</textarea>",
			id, id, text)

	case *CollapsingHeader:
		if !v.init {
			v.cb.pair = [2]string{"[ > ]", "[ < ]"}
			v.cb.OnChange = func() { v.open = !v.open }
			v.cb.Compress()
			v.frame.Header = &v.cb
			v.init = true
		}
		v.cb.Checked = v.open

		chID := fmt.Sprintf("ch_%d", ctx.chC)
		ctx.chC++
		ctx.wm[chID] = v

		arrow := "&#9654;"
		disp := "none"
		if v.open {
			arrow = "&#9660;"
			disp = "block"
		}
		summaryText := esc(textContent(&v.cb))
		content := widgetToHTML(v.root, ctx)
		return fmt.Sprintf("<div><div style=\"cursor:pointer;user-select:none\" onclick=\"fetch('/event',{method:'POST',body:JSON.stringify({type:'widget',id:'%s',action:'toggle'})})\">%s %s</div><div style=\"display:%s;margin-left:16px;border-left:1px solid #666;padding-left:8px;overflow:hidden;max-width:100%%\">%s</div></div>",
			chID, arrow, summaryText, disp, content)

	case *Separator:
		return "<hr style=\"border:none;border-top:1px solid #666;margin:2px 0\">"

	case *Static:
		return widgetToHTML(v.root, ctx)

	case *Text:
		return fmt.Sprintf("<div style=\"color:#000;background:#FFF;white-space:pre-wrap;overflow-wrap:break-word;word-break:break-all\">%s</div>", esc(v.GetText()))

	case *Tree:
		return treeToHTML(v, ctx)

	case *Viewer:
		return viewerToHTML(v, ctx)

	case *Image:
		return imageToHTML(v)

	case *Tabs:
		return tabsToHTML(v, ctx)

	case *ComboBox:
		return comboToHTML(v, ctx)

	case *Menu:
		return menuToHTML(v, ctx)

	case *Stack:
		return widgetToHTML(v.present(), ctx)
	}
	prefix := fmt.Sprintf("u_%d_", ctx.unknownC)
	ctx.unknownC++
	if ctx.w == 0 {
		ctx.w = 80
	}
	cells := renderCells(w, ctx.w)
	return cellsToPre(cells, prefix)
}

func textContent(w Widget) string {
	if w == nil {
		return ""
	}
	switch v := w.(type) {
	case *Text:
		return v.GetText()
	case *Static:
		return textContent(v.root)
	case *CheckBox:
		return v.GetText()
	case *CollapsingHeader:
		return textContent(&v.cb)
	case *Button:
		return v.GetText()
	case *radio:
		return textContent(v.root)
	}
	return ""
}

func isSimpleText(w Widget) bool {
	if w == nil {
		return true
	}
	if _, ok := w.(*Text); ok {
		return true
	}
	if st, ok := w.(*Static); ok {
		_, isText := st.root.(*Text)
		return isText
	}
	return false
}

func radioLabel(r *radio) string {
	if r == nil || r.root == nil {
		return ""
	}
	return textContent(r.root)
}

func viewerToHTML(v *Viewer, ctx *htmlCtx) string {
	prefix := fmt.Sprintf("v_%d_", ctx.vcC)
	ctx.vcC++
	cells := renderCells(v, 2000)
	return cellsToPre(cells, prefix)
}

func imageToHTML(v *Image) string {
	return cellsToPre(v.data, "")
}

func renderCells(w Widget, width uint) [][]Cell {
	h := uint(1000)
	cells := make([][]Cell, h)
	for i := range cells {
		cells[i] = make([]Cell, width)
		for j := range cells[i] {
			cells[i][j] = Cell{S: TextStyle, R: ' '}
		}
	}
	drawer := func(row, col uint, st tcell.Style, r rune) (isVisibleRow bool) {
		if int(row) < len(cells) && int(col) < len(cells[row]) {
			cells[row][col] = Cell{S: st, R: r}
		}
		return int(row) < len(cells)
	}
	w.Render(width, drawer)

	actualH := uint(len(cells))
	for i := len(cells) - 1; i >= 0; i-- {
		nonEmpty := false
		for j := range cells[i] {
			if cells[i][j].R != ' ' {
				nonEmpty = true
				break
			}
		}
		if nonEmpty {
			actualH = uint(i + 1)
			break
		}
	}
	return cells[:actualH]
}

func cellsToPre(cells [][]Cell, classPrefix string) string {
	if len(cells) == 0 {
		return ""
	}
	colorClass := make(map[[2]string]int)
	var nextClass int
	addColor := func(fg, bg string) string {
		key := [2]string{fg, bg}
		if idx, ok := colorClass[key]; ok {
			return fmt.Sprintf("%sc%d", classPrefix, idx)
		}
		idx := nextClass
		nextClass++
		colorClass[key] = idx
		return fmt.Sprintf("%sc%d", classPrefix, idx)
	}

	var buf strings.Builder
	buf.WriteString("<pre style=\"margin:0;white-space:pre-wrap;overflow:hidden;word-wrap:break-word\">")
	for row := range cells {
		for col := range cells[row] {
			fg, bg, _ := cells[row][col].S.Decompose()
			fgHex := tcellColorHex(fg)
			bgHex := tcellColorHex(bg)
			cls := addColor(fgHex, bgHex)
			r := cells[row][col].R
			var ch string
			switch {
			case r == '&':
				ch = "&amp;"
			case r == '<':
				ch = "&lt;"
			case r == '>':
				ch = "&gt;"
			case r == '"':
				ch = "&quot;"
			case r == ' ':
				ch = " "
			case r == 0:
				ch = " "
			default:
				ch = string(r)
			}
			buf.WriteString(fmt.Sprintf("<span class=\"%s\">%s</span>", cls, ch))
		}
		buf.WriteRune('\n')
	}
	buf.WriteString("</pre>")

	var cssBuf strings.Builder
	for key, idx := range colorClass {
		cssBuf.WriteString(fmt.Sprintf(".%sc%d{color:%s;background:%s}\n", classPrefix, idx, key[0], key[1]))
	}
	return "<style>" + cssBuf.String() + "</style>" + buf.String()
}

func treeToHTML(v *Tree, ctx *htmlCtx) string {
	rootHTML := ""
	if v.Root != nil {
		rootHTML = widgetToHTML(v.Root, ctx)
	}
	var children []string
	for i := range v.Nodes {
		children = append(children, treeToHTML(&v.Nodes[i], ctx))
	}
	parts := []string{"<ul style=\"list-style:none;padding-left:16px;margin:0;overflow:hidden;max-width:100%\">"}
	if rootHTML != "" {
		parts = append(parts, fmt.Sprintf("<li>%s", rootHTML))
	}
	for _, c := range children {
		parts = append(parts, fmt.Sprintf("<li>%s</li>", c))
	}
	if rootHTML != "" {
		parts = append(parts, "</li>")
	}
	parts = append(parts, "</ul>")
	return strings.Join(parts, "")
}

func tabsToHTML(v *Tabs, ctx *htmlCtx) string {
	if !v.init {
		v.initialize()
	}

	tabID := fmt.Sprintf("tb_%d", ctx.tabC)
	ctx.tabC++
	ctx.wm[tabID] = v

	pos := int(v.GetPos())

	var headerHTML string
	if v.combo {
		var opts []string
		for i, name := range v.list.names {
			sel := ""
			if i == pos {
				sel = " selected"
			}
			opts = append(opts, fmt.Sprintf("<option value=\"%d\"%s>%s</option>", i, sel, esc(name)))
		}
		headerHTML = fmt.Sprintf("<select onchange=\"fetch('/event',{method:'POST',body:JSON.stringify({type:'widget',id:'%s',action:'tab',index:parseInt(this.value)})})\">%s</select>",
			tabID, strings.Join(opts, ""))
	} else {
		var headers []string
		for i, name := range v.list.names {
			active := ""
			if i == pos {
				active = " background:#FF1493;color:#000"
			}
			idx := i
			headers = append(headers, fmt.Sprintf("<button style=\"border:1px solid #888;cursor:pointer;padding:0 6px;%s\" onclick=\"fetch('/event',{method:'POST',body:JSON.stringify({type:'widget',id:'%s',action:'tab',index:%d})})\">%s</button>",
				active, tabID, idx, esc(name)))
		}
		headerHTML = fmt.Sprintf("<div style=\"display:flex;flex-direction:row;gap:2px;margin-bottom:2px\">%s</div>", strings.Join(headers, ""))
	}

	var content string
	if pos >= 0 && pos < len(v.list.roots) && v.list.roots[pos] != nil {
		content = widgetToHTML(v.list.roots[pos], ctx)
	}

	return fmt.Sprintf("<div>%s<div style=\"border:1px solid #666;padding:4px;min-height:1.2em\">%s</div></div>",
		headerHTML, content)
}

func comboToHTML(v *ComboBox, ctx *htmlCtx) string {
	if !v.init {
		v.rg.OnChange = func() {
			if f := v.OnChange; f != nil {
				f()
			}
		}
		v.rg.OnChange()
		v.init = true
	}

	pos := int(v.GetPos())
	var opts []string
	maxLen := 0
	for i := range v.rg.list.nodes {
		r, ok := v.rg.list.nodes[i].w.(*radio)
		if !ok || r == nil {
			continue
		}
		txt := radioLabel(r)
		if len([]rune(txt)) > maxLen {
			maxLen = len([]rune(txt))
		}
		sel := ""
		if i == pos {
			sel = " selected"
		}
		opts = append(opts, fmt.Sprintf("<option value=\"%d\"%s>%s</option>", i, sel, esc(txt)))
	}
	if len(opts) == 0 {
		opts = append(opts, "<option value=\"\" disabled>---</option>")
	}
	minW := 120
	if maxLen > 0 {
		minW = maxLen * 8 + 20
	}
	id := fmt.Sprintf("sel_%d", ctx.comboE)
	ctx.comboE++
	ctx.wm[id] = v
	return fmt.Sprintf("<select style=\"min-width:%dpx\" onchange=\"fetch('/event',{method:'POST',body:JSON.stringify({type:'widget',id:'%s',action:'select',index:parseInt(this.value)})})\">%s</select>",
		minW, id, strings.Join(opts, ""))
}

func menuToHTML(v *Menu, ctx *htmlCtx) string {
	processMenuState(v)
	isRoot := v.parent == nil

	menuID := fmt.Sprintf("m_%d", ctx.menuC)
	ctx.menuC++
	ctx.wm[menuID] = v

	var subContent string
	for _, sub := range v.subs {
		if sub != nil && sub.opened {
			subContent += fmt.Sprintf("<div style=\"border:1px solid #666;padding:4px;margin-top:2px;background:#FFF\">%s</div>", menuToHTML(sub, ctx))
		}
	}

	if isRoot {
		var buttons []string
		for i, n := range v.header.nodes {
			if n.w == nil {
				continue
			}
			content := esc(textContent(n.w))
			idx := i
			buttons = append(buttons, fmt.Sprintf("<button style=\"border:1px solid #888;cursor:pointer;padding:0 6px\" onclick=\"fetch('/event',{method:'POST',body:JSON.stringify({type:'widget',id:'%s',action:'menu',index:%d})})\">%s</button>",
				menuID, idx, content))
		}
		headerBar := fmt.Sprintf("<div style=\"display:flex;flex-direction:row;gap:2px\">%s</div>", strings.Join(buttons, ""))
		innerHTML := widgetToHTML(v.root, ctx)
		return fmt.Sprintf("<div>%s%s<div style=\"margin-top:2px\">%s</div></div>",
			headerBar, subContent, innerHTML)
	}

	innerHTML := widgetToHTML(&v.list, ctx)
	return fmt.Sprintf("<div>%s%s</div>", innerHTML, subContent)
}

func processMenuState(m *Menu) {
	if m == nil {
		return
	}
	for _, sub := range m.subs {
		processMenuState(sub)
	}
	if m.readyForOpen {
		m.readyForOpen = false
		m.opened = true
	}
}

func mapBrowserKey(key string, ctrl, shift, alt bool) *tcell.EventKey {
	switch key {
	case "Enter":
		return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	case "Backspace":
		return tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone)
	case "Delete":
		return tcell.NewEventKey(tcell.KeyDelete, 0, tcell.ModNone)
	case "ArrowUp":
		return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	case "ArrowDown":
		return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	case "ArrowLeft":
		return tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	case "ArrowRight":
		return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
	case "Tab":
		return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	case "Escape":
		return tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	case "Home":
		return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
	case "End":
		return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
	case "PageUp":
		return tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModNone)
	case "PageDown":
		return tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModNone)
	case " ":
		return tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
	}
	if ctrl {
		switch key {
		case "c", "C":
			return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModCtrl)
		case "a", "A":
			return tcell.NewEventKey(tcell.KeyCtrlA, 0, tcell.ModCtrl)
		case "b", "B":
			return tcell.NewEventKey(tcell.KeyCtrlB, 0, tcell.ModCtrl)
		case "d", "D":
			return tcell.NewEventKey(tcell.KeyCtrlD, 0, tcell.ModCtrl)
		case "e", "E":
			return tcell.NewEventKey(tcell.KeyCtrlE, 0, tcell.ModCtrl)
		case "f", "F":
			return tcell.NewEventKey(tcell.KeyCtrlF, 0, tcell.ModCtrl)
		case "k", "K":
			return tcell.NewEventKey(tcell.KeyCtrlK, 0, tcell.ModCtrl)
		case "n", "N":
			return tcell.NewEventKey(tcell.KeyCtrlN, 0, tcell.ModCtrl)
		case "p", "P":
			return tcell.NewEventKey(tcell.KeyCtrlP, 0, tcell.ModCtrl)
		case "u", "U":
			return tcell.NewEventKey(tcell.KeyCtrlU, 0, tcell.ModCtrl)
		case "w", "W":
			return tcell.NewEventKey(tcell.KeyCtrlW, 0, tcell.ModCtrl)
		case "x", "X":
			return tcell.NewEventKey(tcell.KeyCtrlX, 0, tcell.ModCtrl)
		case "z", "Z":
			return tcell.NewEventKey(tcell.KeyCtrlZ, 0, tcell.ModCtrl)
		}
	}
	runes := []rune(key)
	if len(runes) == 1 && runes[0] >= ' ' {
		return tcell.NewEventKey(tcell.KeyRune, runes[0], tcell.ModNone)
	}
	return nil
}

func (ws *WebServer) generate() (html string) {
	ctx := &htmlCtx{
		w:  ws.width,
		wm: make(map[string]Widget),
	}
	html = widgetToHTML(ws.root, ctx)
	ws.widgetMap = ctx.wm
	return
}

func (ws *WebServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, pageStr)
}

func (ws *WebServer) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var msg eventMsg
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "bad JSON", http.StatusBadRequest)
		return
	}

	ws.mu.Lock()
	defer ws.mu.Unlock()

	switch msg.Type {
	case "resize":
		if msg.Width > 0 {
			ws.width = uint(msg.Width)
		}

	case "widget":
		w, ok := ws.widgetMap[msg.ID]
		if !ok {
			break
		}
		switch msg.Action {
		case "click":
			if btn, ok := w.(*Button); ok && btn.OnClick != nil {
				btn.OnClick()
			}
		case "change":
			if chk, ok := w.(*CheckBox); ok && !chk.ReadOnly {
				chk.Checked = msg.Checked
				if chk.OnChange != nil {
					chk.OnChange()
				}
			}
		case "radio":
			if rg, ok := w.(*RadioGroup); ok {
				rg.SetPos(uint(msg.Index))
			}
		case "input":
			if inp, ok := w.(*InputBox); ok {
				inp.SetText(msg.Text)
			}
		case "select":
			if cb, ok := w.(*ComboBox); ok {
				cb.SetPos(uint(msg.Index))
			}
		case "tab":
			if tabs, ok := w.(*Tabs); ok {
				tabs.SetPos(uint(msg.Index))
			}
		case "menu":
			if m, ok := w.(*Menu); ok {
				m.resetSubmenu()
				if msg.Index >= 0 && msg.Index < len(m.header.nodes) {
					if btn, ok := m.header.nodes[msg.Index].w.(*Button); ok && btn.OnClick != nil {
						btn.OnClick()
					}
				}
			}
		case "toggle":
			if ch, ok := w.(*CollapsingHeader); ok {
				ch.open = !ch.open
			}
		}

	case "key":
		if msg.Ctrl && (msg.Key == "c" || msg.Key == "C") {
			for _, qk := range ws.quitKeys {
				if qk == tcell.KeyCtrlC {
					go ws.shutdown()
					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}
		if ev := mapBrowserKey(msg.Key, msg.Ctrl, msg.Shift, msg.Alt); ev != nil {
			ws.root.Event(ev)
		}

	case "wheel":
		button := tcell.WheelDown
		if msg.Delta < 0 {
			button = tcell.WheelUp
		}
		ev := tcell.NewEventMouse(0, 0, button, tcell.ModNone)
		ws.root.Event(ev)
	}

	html := ws.generate()
	ws.pushSSE(html)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (ws *WebServer) pushSSE(html string) {
	if ws.sse == nil {
		return
	}
	if html == ws.lastHTML {
		return
	}
	ws.lastHTML = html
	msg := sseOutMsg{Type: "full", HTML: html}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case ws.sse.ch <- data:
	default:
	}
}

func (ws *WebServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	conn := &sseConn{
		ch:   make(chan []byte, 8),
		done: make(chan struct{}),
	}

	ws.mu.Lock()
	if ws.sse != nil {
		select {
		case ws.sse.done <- struct{}{}:
		default:
		}
	}
	ws.sse = conn
	ws.lastHTML = ""
	h := ws.generate()
	ws.pushSSE(h)
	ws.mu.Unlock()

	notify := r.Context().Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-notify:
			return
		case <-conn.done:
			return
		case data := <-conn.ch:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func WebRun(root Widget, action chan func(), chQuit <-chan struct{}, quitKeys ...tcell.Key) (err error) {
	if root == nil {
		return fmt.Errorf("root widget is nil")
	}

	ws := &WebServer{
		root:     root,
		action:   action,
		chQuit:   chQuit,
		quitKeys: quitKeys,
		quit:     make(chan struct{}),
		width:    80,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ws.handleRoot)
	mux.HandleFunc("/event", ws.handleEvent)
	mux.HandleFunc("/events", ws.handleSSE)

	ws.srv = &http.Server{Addr: WebAddr, Handler: mux}

	errCh := make(chan error, 1)
	go func() {
		if err := ws.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	for {
		ticker := time.NewTicker(TimeFrameSleep)
		select {
		case err := <-errCh:
			ticker.Stop()
			ws.srv.Close()
			return fmt.Errorf("web: %w", err)
		case <-ws.quit:
			ticker.Stop()
			ws.srv.Close()
			return nil
		case <-chQuit:
			ticker.Stop()
			ws.shutdown()
			ws.srv.Close()
			return nil
		case f := <-action:
			ticker.Stop()
			if f != nil {
				f()
				ws.mu.Lock()
				h := ws.generate()
				ws.pushSSE(h)
				ws.mu.Unlock()
			}
		case <-ticker.C:
			ws.mu.Lock()
			h := ws.generate()
			ws.pushSSE(h)
			ws.mu.Unlock()
		}
	}
}

const pageStr = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#FFF;overflow-y:auto;font-family:monospace;font-size:14px;line-height:1.2;color:#000;padding:4px}
input,textarea,button,select{font-family:inherit;font-size:inherit}
</style>
</head>
<body>
<div id="app"></div>
<script>
(function(){
var es=new EventSource('/events');
var focusedId=null;
var _ss=0,_se=0;
document.addEventListener('input',function(e){
  if(e.target.tagName==='TEXTAREA'){_ss=e.target.selectionStart;_se=e.target.selectionEnd;e.target.style.height='auto';e.target.style.height=e.target.scrollHeight+'px'}
});
document.addEventListener('focusin',function(e){
  if(e.target.tagName==='TEXTAREA'||e.target.tagName==='INPUT'){focusedId=e.target.id}
});
document.addEventListener('focusout',function(e){
  if(e.target.tagName==='TEXTAREA'||e.target.tagName==='INPUT'){focusedId=null}
});
es.onmessage=function(e){
  var m=JSON.parse(e.data);
  var sy=window.scrollY;
  var _st=0;
  if(focusedId){var oe=document.getElementById(focusedId);if(oe&&oe.tagName==='TEXTAREA'){_st=oe.scrollTop}}
  document.getElementById('app').innerHTML=m.html;
  document.querySelectorAll('textarea').forEach(function(ta){ta.style.height='auto';ta.style.height=ta.scrollHeight+'px'});
  window.scrollTo(0,sy);
  if(focusedId){
    var el=document.getElementById(focusedId);
    if(el){el.focus();if(el.tagName==='TEXTAREA'){el.selectionStart=_ss;el.selectionEnd=_se;el.scrollTop=_st}}
  }
};
document.onkeydown=function(e){
  if(e.target&&(e.target.tagName==='TEXTAREA'||e.target.tagName==='INPUT'))return;
  fetch('/event',{method:'POST',body:JSON.stringify({type:'key',key:e.key,code:e.code,ctrl:e.ctrlKey,shift:e.shiftKey,alt:e.altKey})});
};
window.onresize=function(){
  var w=Math.floor(window.innerWidth/8.4);
  if(w<20)w=20;
  fetch('/event',{method:'POST',body:JSON.stringify({type:'resize',width:w})});
};
var w=Math.floor(window.innerWidth/8.4);
fetch('/event',{method:'POST',body:JSON.stringify({type:'resize',width:w||80})});
})();
</script>
</body>
</html>`
