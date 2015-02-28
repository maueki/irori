package main

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
//	"regexp"
	"testing"

	"github.com/zenazn/goji"
	//"github.com/zenazn/goji/web"

	"gopkg.in/mgo.v2"

	"code.google.com/p/go.crypto/bcrypt"
)

func createUser(t *testing.T, db *mgo.Database) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("test"), bcrypt.DefaultCost)
	user := &User{
		Name:        "test",
		Password:    hash,
		Permissions: map[Permission]bool{EDITOR: true},
	}

	err := db.C("user").Insert(user)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTransition(t *testing.T) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	db := session.DB("irori_test")

	db.DropDatabase()

	setRoute(db)

	createUser(t, db)

	s := httptest.NewServer(goji.DefaultMux)
	res, err := http.Get(s.URL + "/wiki")
	if res.StatusCode != http.StatusOK {
		t.Error("GET /wiki unexpected status code: ", res.StatusCode)
	}

	jar, _ := cookiejar.New(nil)
	client := http.Client{
		Jar: jar,
	}

	// Redirect / to /wiki
	res, err = client.Get(s.URL + "/")

	// redirect status code isn't return
	if res.StatusCode != http.StatusOK {
		t.Error("GET / unexpected status code: ", res.StatusCode)
	}

	if res.Request.URL.String() != s.URL+"/wiki" {
		t.Error("GET / unexpected redirect URL: ", res.Request.URL.String())
	}

	res, err = client.Get(s.URL + "/login")
	if res.StatusCode != http.StatusOK {
		t.Error("GET /login unexpected status code:", res.StatusCode)
	}

	// case: login failed
	res, err = client.PostForm(s.URL + "/login",
		url.Values{"username": {"test"}, "password": {"tes"},})
	if res.StatusCode != http.StatusUnauthorized {
		t.Error("POST /login unexpected status code (expected 401): ", res.StatusCode)
	}

	// case: login success
	res, err = client.PostForm(s.URL+"/login",
		url.Values{"username": {"test"}, "password": {"test"}})

	// redirect status code isn't return
	if res.StatusCode != http.StatusOK {
		t.Error("POST /login unexpected status code: ", res.StatusCode)
	}

	if res.Request.URL.String() != s.URL+"/wiki" {
		t.Error("GET / unexpected redirect URL: ", res.Request.URL.String())
	}

	/*
	res, err = client.Get(s.URL + "/action/createNewPage")
	if _, err = regexp.MatchString(`/wiki/[0-9a-f]{24}/edit`, res.Request.URL.String()); err != nil {
		t.Error("Get /action/createNewPage unexpected redirect URL: ", res.Request.URL.String())
		return
	}

	re := regexp.MustCompile(`/wiki/([0-9a-f]{24})/`)
	group := re.FindSubmatch([]byte(res.Request.URL.String()))
	pageurl := s.URL + "/wiki/" + string(group[1])
	res, err = client.PostForm(pageurl,
		url.Values{"title": {"test title"}, "body": {"test body"}})

	if res.StatusCode != http.StatusOK {
		t.Error("POST /wiki/<pageid> unexpected status code: ", res.StatusCode)
	}

	if res.Request.URL.String() != pageurl {
		t.Error("POST /wiki/<pageid> unexpected redirect URL: ", res.Request.URL.String())
	}
*/

}
