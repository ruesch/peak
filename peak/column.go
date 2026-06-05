package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/aleksana/peak/internal/session"
	"github.com/gdamore/tcell/v2"
)

type Gutter struct {
	BaseView
	theme *Theme
}

func (g *Gutter) Layout()                 {}
func (g *Gutter) ShowCursor(tcell.Screen) {}
func (g *Gutter) Draw(s tcell.Screen) {
	sepStyle := tcell.StyleDefault.Background(g.theme.ScrollGutter).Foreground(g.theme.HandleColumn)
	handleStyle := tcell.StyleDefault.Background(g.theme.HandleColumn).Foreground(tcell.ColorBlack)
	for y := g.y; y < g.y+g.h; y++ {
		style := sepStyle
		if y == g.y {
			style = handleStyle
		}
		s.SetContent(g.x, y, ' ', nil, style)
	}
}
func (g *Gutter) Resize(x, y, w, h int) { g.SetPos(x, y, w, h) }

type Column struct {
	TreeNode
	tag           *TextView
	windows       []*Window
	editor        *Editor
	gutter        *Gutter
	onExec        func(*Column, *Window, string) bool
	explicitWidth int
	winCache      []DrawNode
	maximized     *Window
}

func (c *Column) Layout() {}

func (c *Column) PreferredSize() int { return c.explicitWidth }
func (c *Column) MinSize() int       { return 5 }
func (c *Column) SetExplicit(v int)  { c.explicitWidth = v }

func (c *Column) WalkLayout() {
	c.syncChildren()
	c.TreeNode.WalkLayout()
}

func (c *Column) WalkDraw(s tcell.Screen) {
	c.Draw(s)
	c.TreeNode.WalkDraw(s)
}

func (c *Column) Draw(s tcell.Screen) {
	for y := c.y; y < c.y+c.h; y++ {
		for x := c.x + 1; x < c.x+c.w; x++ {
			s.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
	}
}

func (c *Column) ShowCursor(tcell.Screen) {}

func (c *Column) syncChildren() {
	c.children = []DrawNode{c.gutter, c.tag}
	for _, w := range c.windows {
		c.children = append(c.children, w)
	}
}

func NewColumn(x, y, w, h int, editor *Editor, onExec func(*Column, *Window, string) bool) *Column {
	tagStyle := tcell.StyleDefault.Background(editor.theme.ColTagBG).Foreground(editor.theme.ColTagFG)
	tag := NewTextView(" New Zerox Win Delcol ", x+1, y, w-1, 1, tagStyle, true, false)
	tag.style = func() tcell.Style {
		return tcell.StyleDefault.Background(editor.theme.ColTagBG).Foreground(editor.theme.ColTagFG)
	}
	tag.theme = &editor.theme

	gutter := &Gutter{
		BaseView: BaseView{x: x, y: y, w: 1, h: h},
		theme:    &editor.theme,
	}

	c := &Column{
		TreeNode: TreeNode{BaseView: BaseView{x: x, y: y, w: w, h: h}},
		tag:      tag,
		editor:   editor,
		gutter:   gutter,
		onExec:   onExec,
	}
	return c
}

// contentInsertPos scans windows first-to-last and returns the index at which
// a new window should be inserted and how many rows of empty space are available
// there. Empty space is body rows with no content. Returns (len(windows), 0) if
// no window has spare space.
func (c *Column) contentInsertPos() (idx int, emptyH int) {
	for i, win := range c.windows {
		_, total, bodyH := win.body.GetScroll()
		if empty := bodyH - total; empty >= win.MinSize() {
			return i + 1, empty
		}
	}
	return len(c.windows), 0
}

func (c *Column) AddWindow(tagText, bodyText string) *Window {
	if tagText == "" {
		tagText = " ./untitled.txt Get Put Undo Redo Snarf Zerox Del "
	}

	c.maximized = nil
	newWin := NewWindow(tagText, bodyText, c, c.editor, c.x, c.y, c.w, 0, c.onExec)
	newWin.ID = c.editor.nextWinID
	c.editor.nextWinID++

	insertIdx, emptyH := c.contentInsertPos()
	if emptyH > 0 {
		src := c.windows[insertIdx-1]
		src.explicitHeight = src.h - emptyH
		newWin.explicitHeight = emptyH
	}
	c.windows = slices.Insert(c.windows, insertIdx, newWin)

	c.editor.ninep.MountWindow(newWin)
	return newWin
}

func (c *Column) AddTermWindow(tagText, cmd, dir string) (*Window, error) {
	if tagText == "" {
		var name string
		if cmd == "" {
			if name, _ = os.Hostname(); name == "" {
				name = "term"
			}
		} else {
			name = filepath.Base(strings.Fields(cmd)[0])
		}
		tagText = " " + join(dir, "-"+name) + " Zerox Del "
	}

	c.maximized = nil
	newWin, err := NewTermWindow(tagText, c, c.editor, c.x, c.y, c.w, 0, cmd, dir, c.onExec)
	if err != nil {
		return nil, err
	}
	newWin.ID = c.editor.nextWinID
	c.editor.nextWinID++
	c.windows = append(c.windows, newWin)
	c.editor.ninep.MountWindow(newWin)
	return newWin, nil
}

func (c *Column) AddSessionTermWindow(title string, sess session.Session) (*Window, error) {
	c.maximized = nil
	newWin, err := newTermWindowFromSession(" "+title+" Zerox Del ", sess, c, c.editor, c.x, c.y, c.w, 0, c.onExec)
	if err != nil {
		return nil, err
	}
	newWin.ID = c.editor.nextWinID
	c.editor.nextWinID++
	c.windows = append(c.windows, newWin)
	c.editor.ninep.MountWindow(newWin)
	return newWin, nil
}

func (c *Column) Resize(x, y, w, h int) {
	c.SetPos(x, y, w, h)
	c.gutter.Resize(x, y, 1, h)
	c.tag.Resize(x+1, y, w-1, 1)
	if len(c.windows) == 0 {
		return
	}

	if c.maximized != nil {
		// Maximized window fills the column; all others are pushed off-screen below.
		c.maximized.explicitHeight = h - 1
		c.maximized.Resize(x, y+1, w, h-1)
		yOffset := y + h
		for _, win := range c.windows {
			if win != c.maximized {
				wh := win.MinSize()
				win.explicitHeight = wh
				win.Resize(x, yOffset, w, wh)
				yOffset += wh
			}
		}
		return
	}

	availableH := h - 1
	sizes := distribute(c.winNodes(), availableH, c.lastSize)
	c.lastSize = availableH

	yOffset := y + 1
	for i, win := range c.windows {
		win.explicitHeight = sizes[i]
		win.Resize(x, yOffset, w, sizes[i])
		yOffset += sizes[i]
	}
}

func (c *Column) GrowModerate(win *Window) {
	if len(c.windows) <= 1 {
		return
	}
	idx := slices.Index(c.windows, win)
	target := min(win.h+max(5, win.h/2), c.h-1)
	needed := target - win.h
	if needed <= 0 {
		return
	}
	win.explicitHeight = win.h
	for k := 1; k < len(c.windows) && needed > 0; k++ {
		for _, j := range []int{idx + k, idx - k} {
			if j < 0 || j >= len(c.windows) || needed <= 0 {
				continue
			}
			nb := c.windows[j]
			give := min(needed, nb.h-nb.MinSize())
			if give <= 0 {
				continue
			}
			nb.explicitHeight = nb.h - give
			win.explicitHeight += give
			needed -= give
		}
	}
	c.maximized = nil
	c.Resize(c.x, c.y, c.w, c.h)
}

func (c *Column) Maximize(win *Window) {
	idx := slices.Index(c.windows, win)
	if idx > 0 {
		c.windows = slices.Delete(c.windows, idx, idx+1)
		c.windows = slices.Insert(c.windows, 0, win)
	}
	c.maximized = win
	c.Resize(c.x, c.y, c.w, c.h)
}

func (c *Column) GrowFull(win *Window) {
	c.maximized = nil
	avail := c.h - 1
	for _, w := range c.windows {
		if w != win {
			avail -= w.MinSize()
		}
	}
	for _, w := range c.windows {
		if w != win {
			w.explicitHeight = w.MinSize()
		} else {
			w.explicitHeight = max(w.MinSize(), avail)
		}
	}
	c.Resize(c.x, c.y, c.w, c.h)
}

func (c *Column) winNodes() []DrawNode {
	if cap(c.winCache) < len(c.windows) {
		c.winCache = make([]DrawNode, len(c.windows))
	}
	c.winCache = c.winCache[:len(c.windows)]
	for i, w := range c.windows {
		c.winCache[i] = w
	}
	return c.winCache
}

func (c *Column) Contains(x, y int) bool {
	return x >= c.x && x < c.x+c.w && y >= c.y && y < c.y+c.h
}

func (c *Column) HandleEvent(ev tcell.Event) bool {
	if me, ok := ev.(*tcell.EventMouse); ok {
		mx, my := me.Position()
		buttons := me.Buttons()

		if my == c.tag.y {
			if mx == c.x && buttons&(tcell.Button1|tcell.Button2|tcell.Button3) != 0 {
				c.editor.dragCol = c
				c.editor.dragColOrigW = c.explicitWidth
				return false
			}
			if mx > c.x {
				word := c.tag.GetClickWord(mx, my)
				if word != "" {
					if buttons == tcell.Button3 {
						return c.onExec(c, nil, word)
					}
					if buttons == tcell.Button2 {
						return c.editor.Plumb(nil, word)
					}
				}
				if buttons == tcell.Button1 {
					c.editor.dragView, c.editor.focusedView = c.tag, c.tag
				}
				return c.tag.HandleEvent(ev)
			}
		}

		for _, win := range c.windows {
			if win.Contains(mx, my) {
				onHandle := mx == win.x && my >= win.y && my < win.y+win.tagHeight()
				if onHandle && buttons&(tcell.Button1|tcell.Button2|tcell.Button3) != 0 {
					c.editor.dragWin = win
					c.editor.dragWinOrigH = win.explicitHeight
					c.editor.dragWinButton = buttons
					c.editor.dragWinStartY = win.y
					c.editor.ActivateWindow(win)
					c.editor.focusedView = win.tag
					return false
				}
				if buttons == tcell.Button1 {
					c.editor.ActivateWindow(win)
					if my < win.y+win.tagHeight() {
						c.editor.focusedView = win.tag
					}
					c.editor.dragView = c.editor.focusedView
				}
				return win.HandleEvent(ev)
			}
		}
	}
	return false
}
