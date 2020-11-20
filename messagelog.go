package mlog

import (
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/codec"
	"github.com/SlothNinja/color"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

type MLog struct {
	Key        *datastore.Key `datastore:"__key__"`
	Messages   `datastore:"-"`
	SavedState []byte `datastore:",noindex"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (ml *MLog) Load(ps []datastore.Property) error {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	err := datastore.LoadStruct(ml, ps)
	if err != nil {
		return err
	}

	var ms Messages
	err = codec.Decode(&ms, ml.SavedState)
	if err != nil {
		return err
	}
	ml.Messages = ms
	return nil
}

func (ml *MLog) Save() ([]datastore.Property, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	v, err := codec.Encode(ml.Messages)
	if err != nil {
		return nil, err
	}
	ml.SavedState = v
	return datastore.SaveStruct(ml)
}

func (ml *MLog) LoadKey(k *datastore.Key) error {
	ml.Key = k
	return nil
}

type Client struct {
	DS *datastore.Client
}

func NewClient(dsClient *datastore.Client) Client {
	return Client{DS: dsClient}
}

func New(id int64) *MLog {
	return &MLog{Key: datastore.IDKey(kind, id, nil)}
}

const (
	kind     = "MessageLog"
	mlKey    = "MessageLog"
	homePath = "/"
)

func (client Client) AddMessage(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		cu, err := user.CurrentFrom(c)
		if err != nil {
			restful.AddErrorf(c, "must be logged in to send a message")
			c.HTML(http.StatusOK, "shared/flashbox", gin.H{
				"Notices": restful.NoticesFrom(c),
				"Errors":  restful.ErrorsFrom(c),
			})
			return
		}

		ml := From(c)
		if ml == nil {
			log.Errorf("Missing messagelog.")
			restful.AddErrorf(c, "Missing messagelog.")
			c.HTML(http.StatusOK, "shared/flashbox", gin.H{
				"Notices": restful.NoticesFrom(c),
				"Errors":  restful.ErrorsFrom(c),
			})
			return
		}
		m := ml.NewMessage(c)
		m.Text = c.PostForm("message")
		creatorsid := c.PostForm("creatorid")
		if creatorsid != "" {
			intID, err := strconv.ParseInt(creatorsid, 10, 64)
			if err != nil {
				restful.AddErrorf(c, "Invalid value received for creatorsid: %v", creatorsid)
				c.HTML(http.StatusOK, "shared/flashbox", gin.H{
					"Notices": restful.NoticesFrom(c),
					"Errors":  restful.ErrorsFrom(c),
				})
				return
			}
			m.CreatorID = intID
		}
		_, err = client.DS.Put(c, ml.Key, ml)
		if err != nil {
			restful.AddErrorf(c, err.Error())
			log.Errorf(err.Error())
			c.HTML(http.StatusOK, "shared/flashbox", gin.H{
				"Notices": restful.NoticesFrom(c),
				"Errors":  restful.ErrorsFrom(c),
			})
			return
		}
		c.HTML(http.StatusOK, "shared/message", gin.H{
			"message": m,
			"ctx":     c,
			"map":     color.MapFrom(c),
			"link":    cu.Link(),
		})
	}
}

func getID(c *gin.Context) (int64, error) {
	sid := c.Param("hid")
	return strconv.ParseInt(sid, 10, 64)
}

func (client Client) Get(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	id, err := getID(c)
	if err != nil {
		restful.AddErrorf(c, err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	ml := New(id)
	err = client.DS.Get(c, ml.Key, ml)
	if err != nil {
		restful.AddErrorf(c, "Unable to get message log with ID: %v", id)
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}
	with(c, ml)
}

func From(c *gin.Context) (ml *MLog) {
	ml, _ = c.Value(mlKey).(*MLog)
	return
}

func with(c *gin.Context, ml *MLog) *gin.Context {
	c.Set(mlKey, ml)
	return c
}
