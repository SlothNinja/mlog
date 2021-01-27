package mlog

import (
	"errors"
	"html/template"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/codec"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

type MLog struct {
	Key        *datastore.Key `datastore:"__key__"`
	Messages   []*Message     `datastore:"-"`
	SavedState []byte         `datastore:",noindex"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (ml *MLog) Load(ps []datastore.Property) error {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	err := datastore.LoadStruct(ml, ps)
	if err != nil {
		return err
	}

	var ms []*Message
	err = codec.Decode(&ms, ml.SavedState)
	if err != nil {
		return err
	}
	ml.Messages = ms
	return nil
}

func (ml *MLog) Save() ([]datastore.Property, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

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
	*sn.Client
	User *user.Client
}

func NewClient(dsClient *datastore.Client, userClient *user.Client, logger *log.Logger, mcache *cache.Cache) *Client {
	return &Client{
		Client: sn.NewClient(dsClient, logger, mcache, nil),
		User:   userClient,
	}
}

func New(id int64) *MLog {
	return &MLog{Key: key(id)}
}

func key(id int64) *datastore.Key {
	return datastore.IDKey(kind, id, nil)
}

const (
	kind     = "MessageLog"
	mlKey    = "MessageLog"
	msgEnter = "Entering"
	msgExit  = "Exiting"
	homePath = "/"
)

func (ml *MLog) AddMessage(u *user.User, text string) *Message {
	t := time.Now()
	m := &Message{
		CreatorID: u.ID(),
		CreatedAt: t,
		UpdatedAt: t,
		Text:      template.HTMLEscapeString(text),
	}
	ml.Messages = append(ml.Messages, m)
	return m
}

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidCache = errors.New("invalid cached value")
	ErrMissingID    = errors.New("missing identifier")
)

func (client *Client) mcGet(c *gin.Context, id int64) (*MLog, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	k := key(id).Encode()
	item, found := client.Cache.Get(k)
	if !found {
		return nil, ErrNotFound
	}

	ml, ok := item.(*MLog)
	if !ok {
		// delete the invaide cached value
		client.Cache.Delete(k)
		return nil, ErrInvalidCache
	}
	return ml, nil
}

func (client *Client) dsGet(c *gin.Context, id int64) (*MLog, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	ml := New(id)
	err := client.DS.Get(c, ml.Key, ml)
	return ml, err
}

func (client *Client) Get(c *gin.Context, id int64) (*MLog, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	if id == 0 {
		return nil, ErrMissingID
	}

	ml, err := client.mcGet(c, id)
	if err == nil {
		return ml, nil
	}

	return client.dsGet(c, id)
}

func (client *Client) Put(c *gin.Context, id int64, ml *MLog) (*datastore.Key, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	k, err := client.DS.Put(c, key(id), ml)
	if err != nil {
		return nil, err
	}

	return k, client.mcPut(c, k.ID, ml)
}

func (client *Client) mcPut(c *gin.Context, id int64, ml *MLog) error {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	if id == 0 {
		return ErrMissingID
	}

	client.Cache.SetDefault(key(id).Encode(), ml)
	return nil
}
