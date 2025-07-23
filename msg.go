package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Msg struct {
	Id        string    `json:"_id" bson:"_id"`
	Msg       string    `json:"msg" bson:"msg"`
	CreateAt  time.Time `json:"createAt" bson:"createAt"`
	Timestamp string    `json:"timestamp" bson:"-"`
}

// MsgList return list of message info
func MsgList(w http.ResponseWriter, r *http.Request) {
	log.Println("request is:", r.Method, r.RequestURI)
	list := msgList()
	bt, _ := json.Marshal(list)
	w.Write(bt)
}

func msgList() []Msg {
	var list []Msg
	cond := bson.M{}
	err := MClient.Find(cond, &list)
	if err != nil && err != mongo.ErrNilDocument {
		log.Println("find msg error:", err)
		return list
	}
	beijingLocation := time.FixedZone("CST", 8*60*60)
	for idx, l := range list {
		list[idx].Timestamp = l.CreateAt.In(beijingLocation).String()
	}
	return list
}

// CreateMsg get-> message input page
// post-> new message
func CreateMsg(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bt, _ := io.ReadAll(r.Body)
	var msg Msg
	err := json.Unmarshal(bt, &msg)
	if err != nil {
		log.Println("bad format:", len(bt), string(bt))
		w.Write([]byte("bad format: " + err.Error()))
		return
	}
	if len(msg.Msg) > 50000 {
		log.Println("too long:", len(msg.Msg))
		w.Write([]byte("too long"))
		return
	}
	msg.Id = primitive.NewObjectID().Hex()
	msg.CreateAt = time.Now()
	err = MClient.Insert(msg)
	if err != nil {
		log.Println("insert error:", err)
		w.Write([]byte("insert error:" + err.Error()))
		return
	}
	w.Write(bt)
	return
}

func GetMessagePage(w http.ResponseWriter, r *http.Request) {
	str := getMessagePage()
	w.Write([]byte(str))
}
