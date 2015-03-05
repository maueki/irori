package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/flosch/pongo2"
	_ "github.com/flosch/pongo2-addons"
	"github.com/zenazn/goji/web"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Group struct {
	Id    bson.ObjectId `bson:"_id,omitempty" json:"id,omitempty"`
	Name  string
	Users []bson.ObjectId
}

func groupsGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	executeWriterFromFile(w, "view/groups.html", &pongo2.Context{})
}

func apiGroupListGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	wikidb := getWikiDb(c)

	groups := []Group{}

	err := wikidb.Db.C("groups").Find(bson.M{}).All(&groups)
	if err != nil {
		log.Fatal("!!!!! get groups")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(groups)

	if err != nil {
		log.Fatal("!!!!! json.Marshal")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func apiGroupGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	gid := bson.ObjectIdHex(c.URLParams["groupId"])

	if !gid.Valid() {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	wikidb := getWikiDb(c)

	var group Group
	err := wikidb.Db.C("groups").FindId(gid).One(&group)
	if err == mgo.ErrNotFound {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	js, _ := json.Marshal(group)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func apiGroupCreateHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	wikidb := getWikiDb(c)

	defer r.Body.Close()
	var group Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: verify incomming

	changeinfo, err := wikidb.Db.C("groups").Upsert(bson.M{"name": group.Name},
		bson.M{"$setOnInsert": group})
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if changeinfo.UpsertedId == nil {
		w.WriteHeader(http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func apiGroupPutHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	wikidb := getWikiDb(c)

	groupId := bson.ObjectIdHex(c.URLParams["groupId"])

	defer r.Body.Close()
	var group Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: verify incomming

	err := wikidb.Db.C("groups").UpdateId(groupId, group)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	js, _ := json.Marshal(group)
	w.Write(js)
}

func groupEditHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	groupId := c.URLParams["groupId"]

	objid := bson.ObjectIdHex(groupId)
	if !objid.Valid() {
		log.Println("invalid groupId:", groupId)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	executeWriterFromFile(w, "view/edit-group.html", &pongo2.Context{"groupid": groupId})
}

func projectsGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	wikidb := getWikiDb(c)

	projects := []Project{}

	err := wikidb.Db.C("projects").Find(bson.M{}).All(&projects)
	if err != nil {
		log.Fatal("@@@ projects")
	}

	executeWriterFromFile(w, "view/projects.html", &pongo2.Context{"projects": projects})
}

func apiProjectsGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	wikidb := getWikiDb(c)

	projects := []Project{}

	err := wikidb.Db.C("projects").Find(bson.M{}).All(&projects)
	if err != nil {
		log.Fatal("@@@ projects")
	}

	js, err := json.Marshal(projects)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func apiProjectsPostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var p Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	wikidb := getWikiDb(c)

	changeinfo, err := wikidb.Db.C("projects").Upsert(bson.M{"name": p.Name},
		bson.M{"$setOnInsert": p})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if changeinfo.UpsertedId == nil {
		// FIXME: project name already exists.
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func apiPageCreateHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	user := getSessionUser(c)
	wikidb := getWikiDb(c)

	defer r.Body.Close()
	var p Page

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	p.Id = bson.NewObjectId()
	p.Article.Id = bson.NewObjectId()
	p.Article.UserId = user.Id
	p.Article.Date = time.Now()

	log.Println(p)

	err := wikidb.Db.C("pages").Insert(p)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	js, _ := json.Marshal(p)

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func apiPageUpdateHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	user := getSessionUser(c)
	if !user.HasPermission(EDITOR) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	defer r.Body.Close()
	var p Page

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err := p.save(c, r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	js, _ := json.Marshal(p)

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func apiPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	pageId := c.URLParams["pageId"]

	page, err := getPageFromDb(c, pageId)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	js, _ := json.Marshal(page)

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func apiUserListGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	wikidb := getWikiDb(c)

	users := []User{}

	err := wikidb.Db.C("users").Find(bson.M{}).All(&users)
	if err != nil {
		log.Fatal("@@@ users")
	}

	js, err := json.Marshal(users)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

type postedUser struct {
	name     string
	email    string
	password string
}

func apiUserPostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var u postedUser
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if u.email == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user := &User{
		Name:        u.name,
		EMail:       u.email,
		Password:    HashPassword(u.password),
		Permissions: map[Permission]bool{EDITOR: true}, //FIXME
	}

	wikidb := getWikiDb(c)
	// Register user only if user.Email not found.
	changeinfo, err := wikidb.Db.C("users").Upsert(bson.M{"email": user.EMail},
		bson.M{"$setOnInsert": user})
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if changeinfo.UpsertedId == nil {
		log.Println("user.email already exists:", user.EMail)
		w.WriteHeader(http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func userListHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	wikidb := getWikiDb(c)

	users := []User{}

	err := wikidb.Db.C("users").Find(bson.M{}).All(&users)
	if err != nil {
		log.Fatal("!!!! find users", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	executeWriterFromFile(w, "view/users.html", &pongo2.Context{})
}
