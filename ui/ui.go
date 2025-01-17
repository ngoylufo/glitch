package ui

import (
	"fmt"
	"strings"

	"github.com/unitoftime/glitch"
	"github.com/unitoftime/glitch/shaders"
	"github.com/unitoftime/glitch/graph"
)

// Element that needs to be drawn
type uiElement struct {
	drawer Drawer
}

type uiPass struct {
	elements [][]uiElement
}

type uiGlobals struct {
	mouseCaught bool
}
var global uiGlobals

// Must be called every frame before any UI draws happen
// TODO - This is hard to remember to do
func Clear() {
	global.mouseCaught = false
}

func Contains(point glitch.Vec2) bool {
	return global.mouseCaught
}

func mouseCheck(rect glitch.Rect, point glitch.Vec2) bool {
	if global.mouseCaught {
		return false
	}
	if rect.Contains(point[0], point[1]) {
		global.mouseCaught = true
		return true
	}
	return false
}

// TODO do I need an end funtion?
// func End() {
// }

type Drawer interface {
	Bounds() glitch.Rect
	RectDraw(*glitch.RenderPass, glitch.Rect)
	RectDrawColorMask(*glitch.RenderPass, glitch.Rect, glitch.RGBA)
}

type Group struct {
	win *glitch.Window
	pass *glitch.RenderPass
	camera *glitch.CameraOrtho
	atlas *glitch.Atlas
	unionBounds *glitch.Rect // A union of all drawn object's bounds
	allBounds []glitch.Rect
	Debug bool
	OnlyCheckUnion bool
	geomDraw glitch.GeomDraw
	color glitch.RGBA
}

func NewGroup(win *glitch.Window, camera *glitch.CameraOrtho, atlas *glitch.Atlas) *Group {
	shader, err := glitch.NewShader(shaders.SpriteShader)
	if err != nil { panic(err) }
	pass := glitch.NewRenderPass(shader)

	return &Group{
		win: win,
		camera: camera,
		pass: pass,
		atlas: atlas,
		unionBounds: nil,
		allBounds: make([]glitch.Rect, 0),
		Debug: false,
		OnlyCheckUnion: true,
		color: glitch.RGBA{1, 1, 1, 1},
	}
}

// onlyCheckUnion is an optimization if the ui elements are tightly packed (it doesn't loop through each rect
func (g *Group) ContainsMouse() bool {
	if g.OnlyCheckUnion {
		if g.unionBounds == nil { return false }
		return g.unionBounds.Contains(g.mousePosition())
	} else {
		x, y := g.mousePosition()
		for i := range g.allBounds {
			if g.allBounds[i].Contains(x, y) {
				return true
			}
		}
	}
	return false
}

func (g *Group) mousePosition() (float32, float32) {
	x, y := g.win.MousePosition()
	worldSpaceMouse := g.camera.Unproject(glitch.Vec3{x, y, 0})
	return worldSpaceMouse[0], worldSpaceMouse[1]
}

// TODO - Should this be a list of rects that we loop through?
func (g *Group) appendUnionBounds(newBounds glitch.Rect) {
	g.allBounds = append(g.allBounds, newBounds)

	if g.unionBounds == nil {
		g.unionBounds = &newBounds
	} else {
		newUnion := g.unionBounds.Union(newBounds)
		g.unionBounds = &newUnion
	}
}

func (g *Group) Clear() {
	g.pass.Clear()
	g.unionBounds = nil
	g.allBounds = g.allBounds[:0]
}

// Performs a draw of the UI Group
func (g *Group) Draw() {
	g.pass.SetUniform("projection", g.camera.Projection)
	g.pass.SetUniform("view", g.camera.View)

	// Draw the union rect
	if g.Debug {
		if g.unionBounds != nil {
			g.debugRect(*g.unionBounds)
		}
	}

	g.pass.Draw(g.win)
}

func (g *Group) SetColor(color glitch.RGBA) {
	g.color = color
}

func (g *Group) Panel(sprite Drawer, rect glitch.Rect) {
	sprite.RectDrawColorMask(g.pass, rect, g.color)
	g.appendUnionBounds(rect)
	g.debugRect(rect)
}

// Addds a panel with padding to the current bounds of the group
func (g *Group) PanelizeBounds(sprite Drawer, padding glitch.Rect) {
	if g.unionBounds == nil { return }
	rect := *g.unionBounds
	rect = rect.Pad(padding)
	g.pass.SetLayer(128) // Panel layer
	g.Panel(sprite, rect)
	g.pass.SetLayer(glitch.DefaultLayer)
}

func (g *Group) Hover(normal, hovered Drawer, rect glitch.Rect) bool {
	mX, mY := g.mousePosition()
	if !mouseCheck(rect, glitch.Vec2{mX, mY}) {
		g.Panel(hovered, rect)
		return true
	}

	g.Panel(normal, rect)
	return false
}

func (g *Group) Button(normal, hovered, pressed Drawer, rect glitch.Rect) bool {
	mX, mY := g.mousePosition()

	if !mouseCheck(rect, glitch.Vec2{mX, mY}) {
		g.Panel(normal, rect)
		return false
	}

	// If we are here, then we know we are at least hovering
	if g.win.JustPressed(glitch.MouseButtonLeft) {
		g.Panel(pressed, rect)
		return true
	}

	g.Panel(hovered, rect)
	return false
}

// TODO! - text masking around rect?
func (g *Group) Text(str string, rect glitch.Rect, anchor glitch.Vec2) {
	text := g.atlas.Text(str)
	r := rect.Anchor(text.Bounds().ScaledToFit(rect), anchor)
	text.RectDrawColorMask(g.pass, r, g.color)
	g.appendUnionBounds(r)
	g.debugRect(r)
}

// Text, but doesn't automatically scale to fill the rect
// TODO maybe I should call the other text "AutoText"? or something
func (g *Group) FixedText(str string, rect glitch.Rect, anchor glitch.Vec2, scale float32) {
	text := g.atlas.Text(str)
	r := rect.Anchor(text.Bounds().Scaled(scale), anchor)
	text.RectDrawColorMask(g.pass, r, g.color)
	g.appendUnionBounds(r)
	g.debugRect(r)
}

// TODO - combine with fixedtext
func (g *Group) FullFixedText(str string, rect glitch.Rect, anchor, anchor2 glitch.Vec2, scale float32) {
	text := g.atlas.Text(str)
	r := rect.FullAnchor(text.Bounds().Scaled(scale), anchor, anchor2)
	text.RectDrawColorMask(g.pass, r, g.color)
	g.appendUnionBounds(r)
	g.debugRect(r)
}

func (g *Group) TextInput(panel Drawer, str *string, rect glitch.Rect, anchor glitch.Vec2, scale float32) {
	if str == nil { return }

	runes := g.win.Typed()
	*str = *str + string(runes)

	tStr := *str
	if g.win.JustPressed(glitch.KeyBackspace) {
		if g.win.Pressed(glitch.KeyLeftControl) || g.win.Pressed(glitch.KeyRightControl) {
			// Delete whole word
			lastIndex := strings.LastIndex(strings.TrimRight(tStr, " "), " ")
			if lastIndex < 0 {
				// Means there were no spaces, delete everything
				lastIndex = 0
			}

			tStr = tStr[:lastIndex]
		} else {
			if len(tStr) > 0 {
				tStr = tStr[:len(tStr)-1]
			}
		}
	} else if g.win.Repeated(glitch.KeyBackspace) {
		if len(tStr) > 0 {
			tStr = tStr[:len(tStr)-1]
		}
	}

	// ret := false
	// if g.win.JustPressed(glitch.KeyEnter) {
	// 	ret = true
	// }
	*str = tStr

	g.Panel(panel, rect)

	g.FixedText(*str, rect, anchor, scale)
	// return ret
}

// TODO - tooltips only seem to work for single lines
func (g *Group) Tooltip(panel Drawer, tip string, rect glitch.Rect, anchor glitch.Vec2) {
	mX, mY := g.mousePosition()
	if !mouseCheck(rect, glitch.Vec2{mX, mY}) {
		return // If mouse not contained by rect, then don't draw
	}

	text := g.atlas.Text(tip)
	tipRect := rect.Anchor(text.Bounds(), anchor)

	g.Panel(panel, tipRect)

	text.DrawRect(g.pass, tipRect, g.color)
	g.appendUnionBounds(tipRect)
	g.debugRect(tipRect)
}

func (g *Group) LineGraph(rect glitch.Rect, series []glitch.Vec2) {
	line := graph.NewGraph(rect)
	line.Line(series)
	line.Axes()
	line.DrawColorMask(g.pass, glitch.Mat4Ident, g.color)

	g.appendUnionBounds(rect)
	g.debugRect(rect)

	// Draw text around axes
	axes := line.GetAxes()
	g.FullFixedText(fmt.Sprintf("%.2f ms", axes.Min[1]), rect, glitch.Vec2{0, 0}, glitch.Vec2{1, 0.5}, 0.25)
	g.FullFixedText(fmt.Sprintf("%.2f ms", axes.Max[1]), rect, glitch.Vec2{0, 1}, glitch.Vec2{1, 0.5}, 0.25)
}

func (g *Group) debugRect(rect glitch.Rect) {
	if !g.Debug { return }

	lineWidth := float32(2.0)

	g.pass.SetLayer(0) // debug layer

	g.geomDraw.SetColor(glitch.RGBA{1.0, 0, 0, 1.0})
	m := g.geomDraw.Rectangle(rect, lineWidth)
	m.Draw(g.pass, glitch.Mat4Ident)
	g.pass.SetLayer(glitch.DefaultLayer)
}

// func (g *Group) Bar(outer, inner Drawer, bounds glitch.Rect, value float64) Rect {
// 	_, barInner := c.SlicedSprite(outer, bounds)
// 	c.SpriteColorMask(inner, barInner.ScaledXY(barInner.Anchor(0, 0), value, 1), innerColor)
// 	return bounds
// }

// // TODO - tooltips only seem to work for single lines
// func (c *Context) Tooltip(name string, tip string, startRect Rect, position, anchor pixel.Vec) {
// 	padding := 5.0
// 	c.HoverLambda(startRect,
// 		func() {
// 			textBounds := c.MeasureText(tip, position, anchor)
// 			c.SlicedSprite(name, textBounds.Pad(padding, padding))
// 			c.Text(tip, position, anchor)
// 		},
// 		func() {  })
// }

// func (c *Context) BarVert(outer, inner string, bounds Rect, value float64, innerColor color.Color) Rect {
// 	_, barInner := c.SlicedSprite(outer, bounds)
// 	c.SpriteColorMask(inner, barInner.ScaledXY(barInner.Anchor(0, 0), 1, value), innerColor)
// 	return bounds
// }

// func (c *Context) HoverLambda(bounds Rect, ifLambda, elseLambda func()) Rect {
	// mX, mY := g.mousePosition()
// // 	mousePos := c.win.MousePosition()
// 	if bounds.Contains(mousePos) {
// 		ifLambda()
// 	} else {
// 		elseLambda()
// 	}
// 	return bounds
// }

// // TODO - this might have issues with it's bounding box being slightly off because of the rotation
// func (c *Context) SpriteRotated(name string, destRect Rect, radians float64) {
// 	destRect = destRect.Round()

// 	sprite, err := c.spritesheet.Get(name)
// 	if err != nil { panic(err) }
// 	bounds := ZeroRect(sprite.Frame())

// 	scale := pixel.V(destRect.W() / bounds.W(), destRect.H() / bounds.H())
// 	mat := pixel.IM.ScaledXY(pixel.ZV, scale).Rotated(pixel.ZV, radians)
// 	mat = mat.Moved(destRect.Center())

// 	c.appendUnionBounds(destRect)
// 	sprite.Draw(c.batch, mat)

// 	c.DebugRect(destRect)
// }
