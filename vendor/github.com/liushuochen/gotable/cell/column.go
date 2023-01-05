package cell

import (
	"github.com/liushuochen/gotable/color"
	"github.com/liushuochen/gotable/util"
)

const (
	AlignCenter = iota
	AlignLeft
	AlignRight
)

type Column struct {
	name         string
	coloredName  string
	defaultValue string
	align        int
	length       int
}

func CreateColumn(name string) *Column {
	h := &Column{
		name:         name,
		coloredName:  name,
		defaultValue: "",
		align:        AlignCenter,
		length:       util.Length(name),
	}
	return h
}

func (h *Column) String() string {
	return h.coloredName
}

func (h *Column) Original() string {
	return h.name
}

func (h *Column) Length() int {
	return h.length
}

func (h *Column) Default() string {
	return h.defaultValue
}

func (h *Column) SetDefault(value string) {
	h.defaultValue = value
}

func (h *Column) Align() int {
	return h.align
}

func (h *Column) AlignString() string {
	switch h.align {
	case AlignCenter:
		return "center"
	case AlignLeft:
		return "left"
	case AlignRight:
		return "right"
	default:
		return "unknown"
	}
}

func (h *Column) SetAlign(mode int) {
	switch mode {
	case AlignLeft:
		h.align = AlignLeft
	case AlignRight:
		h.align = AlignRight
	default:
		h.align = AlignCenter
	}
}

func (h *Column) Equal(other *Column) bool {
	functions := []func(o *Column) bool{
		h.nameEqual,
		h.lengthEqual,
		h.defaultEqual,
		h.alignEqual,
	}

	for _, function := range functions {
		if !function(other) {
			return false
		}
	}
	return true
}

func (h *Column) SetColor(displayType, font, background int) {
	c := new(color.Color)
	c.Display = displayType
	c.Font = font
	c.Background = background
	h.coloredName = c.Combine(h.Original())
	return
}

func (h *Column) Colorful() bool {
	return h.String() != h.Original()
}

func (h *Column) nameEqual(other *Column) bool {
	return h.Original() == other.Original()
}

func (h *Column) lengthEqual(other *Column) bool {
	return h.Length() == other.Length()
}

func (h *Column) defaultEqual(other *Column) bool {
	return h.Default() == other.Default()
}

func (h *Column) alignEqual(other *Column) bool {
	return h.Align() == other.Align()
}
