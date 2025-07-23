package main

import (
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Msg struct {
	Id       string    `json:"_id" bson:"_id"`
	Msg      string    `json:"msg" bson:"msg"`
	CreateAt time.Time `json:"createAt" bson:"createAt"`
}

// MsgList return list of message info
// get only
func MsgList(w http.ResponseWriter, r *http.Request) {
	log.Println("request is:", r.Method, r.RequestURI)
	if r.Method != http.MethodGet {
		log.Println("bad method for msg list:", r.Method)
		w.Write([]byte("bad method for msg list"))
		return
	}
	var list []Msg
	cond := bson.M{}
	err := MClient.Find(cond, &list)
	if err != nil && err != mongo.ErrNilDocument {
		log.Println("find msg error:", err)
		w.Write([]byte("find msg error: " + err.Error()))
		return
	}
	ht := "<html><head><title>Msg List</title></head><body><h1><div align=\"center\">"
	if len(list) == 0 {
		ht = ht + "Empty List"
	}
	beijingLocation := time.FixedZone("CST", 8*60*60)
	for _, l := range list {
		ht = ht + "<a href=\"/t/list/" + l.Id + "\"> Info </a>, " + l.CreateAt.In(beijingLocation).String() + "<br>"
	}
	ht = ht + "</div></h1></body></html>"
	w.Write([]byte(ht))
}

// MsgShow return single message
// get only
func MsgShow(w http.ResponseWriter, r *http.Request) {
	log.Println("request is:", r.Method, r.RequestURI)
	if r.Method != http.MethodGet {
		log.Println("bad method for msg show:", r.Method)
		w.Write([]byte("bad method for msg show"))
		return
	}
	msgId := r.RequestURI[len("/t/list/"):]
	log.Println("msg show id:", msgId)
	cond := bson.M{
		"_id": msgId,
	}
	var msg []Msg
	err := MClient.Find(cond, &msg)
	if err != nil || len(msg) <= 0 {
		log.Println("msg id:", msgId, err)
		w.Write([]byte("no such msg id:" + msgId))
		return
	}
	w.Write([]byte(msg[0].Msg))
}

// CreateMsg get-> message input page
// post-> new message
func CreateMsg(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		msg := r.Form["message"]
		message := ""
		if len(msg) > 0 {
			message = msg[0]
		}
		if len(message) > 500000 || len(message) < 2 {
			log.Println("empty msg form is:", r.Form)
			w.Write([]byte("too long or short"))
			return
		}
		var m Msg
		m.Id = primitive.NewObjectID().Hex()
		m.CreateAt = time.Now()
		m.Msg = message
		err := MClient.Insert(m)
		if err != nil {
			w.Write([]byte(err.Error()))
		} else {
			http.Redirect(w, r, "/t/list/"+m.Id, 302)
			return
		}
		return
	}
	log.Println("get /t")
	a, b := w.Write([]byte(GetMessagePage()))
	log.Println("write done:", a, b)
	return
}
