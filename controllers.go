package main

import (
	"crypto/sha1"
	"encoding/json"
	"image/color"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/flosch/pongo2"
	_ "github.com/flosch/pongo2-addons"
	"github.com/zenazn/goji/web"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/cupcake/sigil/gen"

	"code.google.com/p/go.crypto/bcrypt"
)

type group struct {
	Id    bson.ObjectId `bson:"_id,omitempty" json:"id,omitempty"`
	Name  string
	Users []bson.ObjectId
}

func groupsGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	executeWriterFromFile(w, "view/groups.html", &pongo2.Context{})
}

func apiGroupListGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	docdb := getDocDb(c)

	groups := []group{}

	err := docdb.Db.C("groups").Find(bson.M{}).All(&groups)
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

	docdb := getDocDb(c)

	var group group
	err := docdb.Db.C("groups").FindId(gid).One(&group)
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
	docdb := getDocDb(c)

	defer r.Body.Close()
	var group group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: verify incomming

	changeinfo, err := docdb.Db.C("groups").Upsert(bson.M{"name": group.Name},
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
	docdb := getDocDb(c)

	groupId := bson.ObjectIdHex(c.URLParams["groupId"])

	defer r.Body.Close()
	var group group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: verify incomming

	err := docdb.Db.C("groups").UpdateId(groupId, group)
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
	docdb := getDocDb(c)

	projects := []project{}

	err := docdb.Db.C("projects").Find(bson.M{}).All(&projects)
	if err != nil {
		log.Fatal("@@@ projects")
	}

	executeWriterFromFile(w, "view/projects.html", &pongo2.Context{"projects": projects})
}

func apiProjectsGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	docdb := getDocDb(c)

	projects := []project{}

	err := docdb.Db.C("projects").Find(bson.M{}).All(&projects)
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
	var p project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	docdb := getDocDb(c)

	changeinfo, err := docdb.Db.C("projects").Upsert(bson.M{"name": p.Name},
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
	docdb := getDocDb(c)

	defer r.Body.Close()
	var p page

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	p.Id = bson.NewObjectId()
	p.Author = user.Id
	p.Article.Id = bson.NewObjectId()
	p.Article.UserId = user.Id
	p.Article.Date = time.Now()

	log.Println(p)

	err := docdb.Db.C("pages").Insert(p)
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
	var p page

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

func apiOwnPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	user := getSessionUser(c)
	docdb := getDocDb(c)

	var pages []page

	err := docdb.Db.C("pages").Find(bson.M{"author": user.Id}).Select(bson.M{"history": 0}).All(&pages)
	if err != nil {
		log.Println("apiPageListGetHandler Find Failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(pages)
	if err != nil {
		log.Println("apiPageListGetHandler json Marshal Failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func apiPageListGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	docdb := getDocDb(c)

	var pages []page

	err := docdb.Db.C("pages").Find(bson.M{}).Select(bson.M{"history": 0}).All(&pages)
	if err != nil {
		log.Println("apiPageListGetHandler Find Failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(pages)
	if err != nil {
		log.Println("apiPageListGetHandler json Marshal Failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func apiUserListGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	docdb := getDocDb(c)

	users := []user{}

	err := docdb.Db.C("users").Find(bson.M{"$or": []interface{}{
		bson.M{"disabled": bson.M{"$exists": false}},
		bson.M{"disabled": false}}}).All(&users)

	if err != nil {
		log.Println("apiUserListGetHandler: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(users)
	if err != nil {
		log.Println("apiUserListGetHandler: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func apiUserDeleteHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	uid := bson.ObjectIdHex(c.URLParams["userId"])
	if !uid.Valid() {
		log.Println("uid invalid: ", uid)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	docdb := getDocDb(c)

	err := docdb.Db.C("users").UpdateId(uid, bson.M{"disabled": true})
	if err == mgo.ErrNotFound {
		log.Println("user not found: ", uid)
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("user delete failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	return
}

type postedUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func apiUserPostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var u postedUser
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// FIXME: veirfy
	if len(u.Name) < 4 || u.Email == "" {
		log.Println("user is imcomplete, ", u)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(u.Password) < 8 {
		log.Println("password is too short.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user := &user{
		Name:        u.Name,
		EMail:       u.Email,
		Password:    HashPassword(u.Password),
		Permissions: map[permission]bool{EDITOR: true}, //FIXME
		Disabled:    false,
	}

	docdb := getDocDb(c)
	// Register user only if user.Email not found.
	changeinfo, err := docdb.Db.C("users").Upsert(bson.M{"email": user.EMail},
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
	docdb := getDocDb(c)

	users := []user{}

	err := docdb.Db.C("users").Find(bson.M{}).All(&users)
	if err != nil {
		log.Fatal("!!!! find users", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	executeWriterFromFile(w, "view/users.html", &pongo2.Context{})
}

var config = gen.Sigil{
	Rows: 5,
	Foreground: []color.NRGBA{
		rgb(45, 79, 255),
		rgb(254, 180, 44),
		rgb(226, 121, 234),
		rgb(30, 179, 253),
		rgb(232, 77, 65),
		rgb(49, 203, 115),
		rgb(141, 69, 170),
	},
	Background: rgb(224, 224, 224),
}

func rgb(r, g, b uint8) color.NRGBA { return color.NRGBA{r, g, b, 255} }

func apiUserIconHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	userid := c.URLParams["userId"]

	// FIXME: use identicon only if user dosen't have icon.
	h := sha1.New()
	io.WriteString(h, userid)

	w.Header().Set("Content-Type", "image/svg+xml")
	config.MakeSVG(w, 250, false, h.Sum(nil))
}

type updatePassword struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func apiPasswordHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	user := getSessionUser(c)
	defer r.Body.Close()

	var p updatePassword
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := bcrypt.CompareHashAndPassword(user.Password, []byte(p.CurrentPassword))
	if err != nil {
		log.Println("apiPasswordHandler Failed: Password Incorrect")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	docdb := getDocDb(c)
	err = docdb.Db.C("users").UpdateId(user.Id, bson.M{"$set": bson.M{"password": HashPassword(p.NewPassword)}})
	if err != nil {
		log.Println("apiPasswordHandler Failed: Update password")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
