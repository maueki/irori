package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coopernurse/gorp"
	_ "github.com/mattn/go-sqlite3"
	"github.com/zenazn/goji/web"

	"gopkg.in/mgo.v2"
)

func testDb(dbmap *gorp.DbMap, mongodb *mgo.Database) func(c *web.C, h http.Handler) http.Handler {
	wikidb := &WikiDb{
		DbMap: dbmap,
		Db:    mongodb,
	}

	return func(c *web.C, h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			c.Env = map[string]interface{}{"wikidb": wikidb}
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
	mongodb := session.DB("gowikitest")

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal("sql.Open failed")
	}
	defer db.Close()

	dbmap, err := createTable(db)
	if err != nil {
		t.Fatal("CreateTable Failed")
	}

	m := web.New()
	m.Use(testDb(dbmap, mongodb))
	m.Get("/wiki", topPageGetHandler)
	s := httptest.NewServer(m)
	defer s.Close()

	res, err := http.Get(s.URL + "/wiki")
	if res.StatusCode != http.StatusOK {
		t.Error("unexpected", res.StatusCode)
	}
}
