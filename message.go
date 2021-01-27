package mlog

import (
	"html/template"
	"time"

	"github.com/SlothNinja/color"
)

type Message struct {
	Text      string
	CreatorID int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (m *Message) Color(cm color.Map) template.HTML {
	c, ok := cm[int(m.CreatorID)]
	if !ok {
		return template.HTML("default")
	}
	return template.HTML(c.String())
}

func (m *Message) Message() template.HTML {
	return template.HTML(template.HTMLEscapeString(m.Text))
}
