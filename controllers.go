package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/flosch/pongo2"
	_ "github.com/flosch/pongo2-addons"
	"github.com/zenazn/goji/web"

	"gopkg.in/mgo.v2/bson"
)

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
