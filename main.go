package main

import (
	"fmt"
	"log"
	"net/http"
	"database/sql"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"github.com/flosch/pongo2"
	"github.com/coopernurse/gorp"
	_ "github.com/mattn/go-sqlite3"

	//"github.com/gorilla/sessions"
)

type Page struct {
	Id      int64 `db:"post_id"`
	Title string
	Body string
}

type User struct {
	Id      int64 `db:"user_id"`
	Name    string `db:"name"`
	Password string `db: "password"` // FIXME: plain text
}

type WikiDb struct {
	DbMap *gorp.DbMap
}

func (p *Page) save(c web.C) error {
	wikidb := getWikiDb(c)
	pOld := Page{}
	err := wikidb.DbMap.SelectOne(&pOld, "select * from page where title=?", p.Title)
	if err == sql.ErrNoRows {
		return wikidb.DbMap.Insert(p)
	} else if err != nil {
		log.Fatalln(err)
	}
	p.Id = pOld.Id
	_, err = wikidb.DbMap.Update(p)
	return err
}

func loadPage(c web.C, title string) (*Page, error) {
	wikidb := getWikiDb(c)
	p := Page{}
	err := wikidb.DbMap.SelectOne(&p, "select * from page where title=?", title)
	if err != nil {
		return nil, err
	}
	return &p,nil
}

func viewHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	title := c.URLParams["title"]
	p, _ := loadPage(c, title)
	fmt.Fprintf(w, "<h1>%s</h1><div>%s</div>", p.Title, p.Body)
}

var editTpl = pongo2.Must(pongo2.FromFile("view/edit.html"))

func editHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	title := c.URLParams["title"]
	p, err := loadPage(c, title)
	if err != nil {
		p = &Page{Title: title}
	}

	err = editTpl.ExecuteWriter(pongo2.Context{"page": p}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func saveHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	title := c.URLParams["title"]
	body := r.FormValue("body")
	p := &Page{Title: title, Body: body}
	err := p.save(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/wiki/" + title, http.StatusFound)
}

func getWikiDb(c web.C) *WikiDb {
	return c.Env["wikidb"].(*WikiDb)
}

var signupTpl = pongo2.Must(pongo2.FromFile("view/signup.html"))

func signupHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	err := signupTpl.ExecuteWriter(pongo2.Context{}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func signupPostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	wikidb := getWikiDb(c)
	name := r.FormValue("username")
	password := r.FormValue("password")

	user := &User{Name: name, Password: password}
	err := wikidb.DbMap.Insert(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	http.Redirect(w, r, "/signup", http.StatusFound)
}

func createTable(db *sql.DB) (*gorp.DbMap, error) {
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	pageTable := dbmap.AddTableWithName(Page{}, "page").SetKeys(true, "Id")
	pageTable.ColMap("Title").Rename("title")
	pageTable.ColMap("Body").Rename("body")

	userTable := dbmap.AddTableWithName(User{}, "user").SetKeys(true, "Id")
	userTable.ColMap("Name").SetUnique(true)

	dbmap.DropTables()
	err := dbmap.CreateTables()
	if err != nil {
		return nil, err
	}

	return dbmap,err
}

func includeDb(dbmap *gorp.DbMap) func(c *web.C, h http.Handler) http.Handler {
	wikidb := &WikiDb{
		DbMap : dbmap,
	}

	return func(c *web.C, h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			c.Env["wikidb"] = wikidb
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func main() {
	db, err := sql.Open("sqlite3", "./wiki.db")
	if err != nil {
		log.Fatalln(err)
	}

	dbmap, err := createTable(db)
	if err != nil {
		log.Fatalln(err)
	}
	defer dbmap.Db.Close()

	goji.Get("/signup", signupHandler)

	goji.Use(includeDb(dbmap))

	goji.Post("/signup", signupPostHandler)

	goji.Get("/wiki/:title", viewHandler)
	goji.Get("/wiki/:title/edit", editHandler)
	goji.Post("/wiki/:title", saveHandler)
	goji.Serve()
}
