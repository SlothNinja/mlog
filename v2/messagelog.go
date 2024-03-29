package mlog

import (
	"encoding/json"
	"errors"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/codec"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/sn/v2"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

type MLog struct {
	Key        *datastore.Key `datastore:"__key__"`
	Messages   []*Message     `datastore:"-"`
	Read       map[int64]int  `datastore:"-" json:"read"`
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

	obj := struct {
		Messages []*Message    `json:"messages"`
		Read     map[int64]int `json:"read"`
	}{}

	err = json.Unmarshal(ml.SavedState, &obj)
	if err != nil {
		var ms []*Message
		err = codec.Decode(&ms, ml.SavedState)
		if err != nil {
			return err
		}
		ml.Messages = ms
		ml.Read = make(map[int64]int)
		return nil
	}
	ml.Messages = obj.Messages
	ml.Read = obj.Read
	if ml.Read == nil {
		ml.Read = make(map[int64]int)
	}
	return nil
}

func (ml *MLog) Save() ([]datastore.Property, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	obj := struct {
		Messages []*Message    `json:"messages"`
		Read     map[int64]int `json:"read"`
	}{Messages: ml.Messages, Read: ml.Read}

	v, err := json.Marshal(&obj)
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

func NewClient(snClient *sn.Client, userClient *user.Client) *Client {
	return &Client{
		Client: snClient,
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
	m := NewMessage(u, text)
	ml.Messages = append(ml.Messages, m)
	if ml.Read == nil {
		ml.Read = make(map[int64]int)
	}
	ml.Read[u.ID()] = len(ml.Messages)
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
		return ml, err
	}

	return client.dsGet(c, id)
}

func (client *Client) UpdateRead(c *gin.Context, ml *MLog, u *user.User) (*MLog, error) {
	ml.Read[u.ID()] = len(ml.Messages)
	_, err := client.Put(c, ml.Key.ID, ml)
	if err != nil {
		return nil, err
	}
	return ml, nil
}

func (client *Client) Unread(c *gin.Context, id int64, u *user.User) (int, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	if id == 0 {
		return -1, ErrMissingID
	}

	ml, err := client.mcGet(c, id)
	if err == nil {
		client.Log.Debugf("mcGet ml.Read: %#v", ml.Read)
		client.Log.Debugf("mcGet len(ml.Messages): %v", len(ml.Messages))
		return len(ml.Messages) - ml.Read[u.ID()], nil
	}

	ml, err = client.dsGet(c, id)
	if err != nil {
		return -1, err
	}
	client.Log.Debugf("dsGet ml.Read: %#v", ml.Read)
	client.Log.Debugf("dsGet len(ml.Messages): %v", len(ml.Messages))
	return len(ml.Messages) - ml.Read[u.ID()], nil
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
