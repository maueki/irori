package main

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"gopkg.in/mgo.v2"
)

func testDb(db *mgo.Database) func(c *web.C, h http.Handler) http.Handler {
	wikidb := &WikiDb{
		Db: db,
	}

	return func(c *web.C, h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			c.Env = map[interface{}]interface{}{"wikidb": wikidb}
			//c.Env["wikidb"] = wikidb
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
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

	res, err = client.Get(s.URL + "/signup")
	if res.StatusCode != http.StatusOK {
		t.Error("POST /signup unexpected status code", res.StatusCode)
	}

	res, err = client.Get(s.URL + "/login")
	if res.StatusCode != http.StatusOK {
		t.Error("GET /login unexpected status code:", res.StatusCode)
	}

	// case: login failed
	res, err = client.PostForm(s.URL+"/login",
		url.Values{"username": {"test"}, "password": {"tes"}})
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

}
