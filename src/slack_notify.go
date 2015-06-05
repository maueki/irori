package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type pageHookSlack struct {
	db *mgo.Database
}

type SlackNotify struct {
	Text string `json:"text"`
}

type slackSettings struct {
	Id      bson.ObjectId `bson:"_id, omitempty"`
	Url     string        `bson:"url"`
	Project bson.ObjectId `bson:"project"`
}

func storeSlackSetting(slackurl string, p project) {
	log.Println(slackurl, p)
}

func (hook pageHookSlack) sendNotify(msg string, p page) {
	portstr := ""
	if IroriConfig.Port != 0 {
		portstr = ":" + strconv.Itoa(IroriConfig.Port)
	}

	pageurl := fmt.Sprintf("http://%s%s/docs/%s", IroriConfig.HostName, portstr, p.Id.Hex())

	notify := SlackNotify{
		Text: fmt.Sprintf("%s\n\n<%s|%s>",
			msg, pageurl, p.Article.Title)}

	js, _ := json.Marshal(notify)

	var projects []project
	hook.db.C("projects").Find(bson.M{"_id": bson.M{"$in": p.Projects}}).All(&projects)
	for _, proj := range projects {
		if proj.SlackURL == "" {
			continue
		}

		req, _ := http.NewRequest("POST", proj.SlackURL, bytes.NewBuffer(js))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println("Failed to send json")
		} else if resp != nil {
			defer resp.Body.Close()
		}
	}

}

func (hook pageHookSlack) onCreate(p page) {
	user, err := getUserById(hook.db, p.Author)

	if err != nil {
		log.Println("SlackHook onCreate: ", err)
		return
	}

	msg := fmt.Sprintf("記事が%sにより投稿されました", user.Name)

	hook.sendNotify(msg, p)
}

func (hook pageHookSlack) onUpdate(p page) {
	user, err := getUserById(hook.db, p.Article.UserId)

	if err != nil {
		log.Println("SlackHook onUpdate: ", err)
		return
	}

	msg := fmt.Sprintf("記事が%sにより編集されました", user.Name)

	hook.sendNotify(msg, p)
}
