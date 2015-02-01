package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coopernurse/gorp"
	_ "github.com/mattn/go-sqlite3"
	"github.com/zenazn/goji/web"
)

func testDb(dbmap *gorp.DbMap) func(c *web.C, h http.Handler) http.Handler {
	wikidb := &WikiDb{
		DbMap: dbmap,
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
	m.Use(testDb(dbmap))
	m.Get("/wiki", topPageGetHandler)
	s := httptest.NewServer(m)
	defer s.Close()

	res, err := http.Get(s.URL + "/wiki")
	if res.StatusCode != http.StatusOK {
		t.Error("unexpected", res.StatusCode)
	}
}
