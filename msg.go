package main

import (
	"encoding/json"
	"html"
	"log"
	"net/http"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Msg struct {
	Id       string    `json:"_id" bson:"_id"`
	Msg      string    `json:"msg" bson:"msg"`
	CreateAt time.Time `json:"createAt" bson:"createAt"`
}

type MsgPreview struct {
	Id        string    `json:"_id"`
	Preview   string    `json:"preview"`
	Len       int       `json:"len"`
	Truncated bool      `json:"truncated"`
	CreateAt  time.Time `json:"createAt"`
}

// msgPreview cut a message down for the card list without splitting a rune.
func msgPreview(s string) string {
	const maxBytes = 300
	if len(s) <= maxBytes {
		return s
	}
	cut := s[:maxBytes]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut
}

func writeJson(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	bt, err := json.Marshal(v)
	if err != nil {
		log.Println("marshal json error:", err)
		return
	}
	w.Write(bt)
}

// ApiMsgList return message list as json, newest first
// get only
func ApiMsgList(w http.ResponseWriter, r *http.Request) {
	log.Println("request is:", r.Method, r.RequestURI)
	if r.Method != http.MethodGet {
		http.Error(w, "bad method for msg list", http.StatusMethodNotAllowed)
		return
	}
	var list []Msg
	cond := bson.M{}
	sort := bson.M{"createAt": -1}
	err := MClient.FindSort(cond, sort, &list)
	if err != nil && err != mongo.ErrNilDocument {
		log.Println("find msg error:", err)
		http.Error(w, "find msg error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var previews []MsgPreview
	for _, l := range list {
		previews = append(previews, MsgPreview{
			Id:        l.Id,
			Preview:   msgPreview(l.Msg),
			Len:       len(l.Msg),
			Truncated: len(l.Msg) > 300,
			CreateAt:  l.CreateAt,
		})
	}
	if previews == nil {
		previews = []MsgPreview{}
	}
	writeJson(w, bson.M{"list": previews})
}

// ApiMsgCreate create message from json body
// post only
func ApiMsgCreate(w http.ResponseWriter, r *http.Request) {
	log.Println("request is:", r.Method, r.RequestURI)
	if r.Method != http.MethodPost {
		http.Error(w, "bad method for msg create", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Msg string `json:"msg"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		log.Println("decode msg body error:", err)
		http.Error(w, "bad json body", http.StatusBadRequest)
		return
	}
	if len(body.Msg) > 500000 || len(body.Msg) < 2 {
		http.Error(w, "too long or short", http.StatusBadRequest)
		return
	}
	var m Msg
	m.Id = primitive.NewObjectID().Hex()
	m.CreateAt = time.Now()
	m.Msg = body.Msg
	err = MClient.Insert(m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJson(w, bson.M{"id": m.Id})
}

// ApiMsgDelete delete one message by id
// post only
func ApiMsgDelete(w http.ResponseWriter, r *http.Request) {
	log.Println("request is:", r.Method, r.RequestURI)
	if r.Method != http.MethodPost {
		http.Error(w, "bad method for msg delete", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Id string `json:"id"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil || body.Id == "" {
		http.Error(w, "bad json body", http.StatusBadRequest)
		return
	}
	cond := bson.M{
		"_id": body.Id,
	}
	err = MClient.Del(cond)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJson(w, bson.M{"ok": true})
}

// MsgListRedirect keep old /t/list links alive by sending them to /t
func MsgListRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/t", http.StatusFound)
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
	if len(msgId) < len("6874bfcd3a56fd69fb514e37") {
		http.Redirect(w, r, "/t", http.StatusFound)
		return
	}
	log.Println("msg show id:", msgId)
	cond := bson.M{
		"_id": msgId,
	}
	var msg []Msg
	err := MClient.Find(cond, &msg)
	if err != nil || len(msg) <= 0 {
		log.Println("msg id:", msgId, err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("no such msg id:" + html.EscapeString(msgId)))
		return
	}
	// Serve as plain text to avoid stored XSS when message contains HTML/JS.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write([]byte(msg[0].Msg))
}

// CreateMsg get-> message page
// post-> new message, keep form + redirect for curl compatibility
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
