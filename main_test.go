package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestSample(t *testing.T) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	db := session.DB("gowikitest")

	m := web.New()
	m.Use(testDb(db))
	m.Get("/wiki", topPageGetHandler)
	s := httptest.NewServer(m)
	defer s.Close()

	res, err := http.Get(s.URL + "/wiki")
	if res.StatusCode != http.StatusOK {
		t.Error("unexpected", res.StatusCode)
	}
}
