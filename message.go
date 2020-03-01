package mlog

import (
	"html/template"
	"time"

	"github.com/SlothNinja/color"
	"github.com/gin-gonic/gin"
)

type Message struct {
	Text      string
	CreatorID int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ml *MLog) NewMessage(c *gin.Context) *Message {
	t := time.Now()
	m := &Message{
		CreatedAt: t,
		UpdatedAt: t,
	}
	ml.Messages = append(ml.Messages, m)
	return m
}

type Messages []*Message

//func (m *Message) UserLink(ctx context.Context) (ul template.HTML) {
//	creator := user.New(ctx)
//	creator.ID = m.CreatorID
//	switch err := datastore.Get(ctx, creator); {
//	case err != nil:
//		log.Errorf(ctx, "Message#UserLink Error: %v", err)
//	case creator == nil:
//		log.Errorf(ctx, "Unable to find creator for ID: %v", m.CreatorID)
//	default:
//		ul = creator.Link()
//	}
//	return
//}

func (m *Message) Color(cm color.Map) template.HTML {
	if c, ok := cm[int(m.CreatorID)]; ok {
		return template.HTML(c.String())
	}
	return template.HTML("default")
}

func (m *Message) Message() template.HTML {
	return template.HTML(template.HTMLEscapeString(m.Text))
}
