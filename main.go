package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"code.google.com/p/go.crypto/bcrypt"

	"github.com/coopernurse/gorp"
	"github.com/flosch/pongo2"
	_ "github.com/flosch/pongo2-addons"
	_ "github.com/mattn/go-sqlite3"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"github.com/gorilla/sessions"
)

const SESSION_NAME = "go_wiki_session"

var store = sessions.NewCookieStore([]byte("something-very-secret")) // FIXME

type Page struct {
	Id    int64 `db:"post_id"`
	Title string
	Body  string
}

type User struct {
	Id       int64  `db:"user_id"`
	Name     string `db:"name"`
	Password []byte `db:"password"`
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
	return &p, nil
}

func viewHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	viewTpl := pongo2.Must(pongo2.FromFile("view/view.html"))
	title := c.URLParams["title"]
	p, _ := loadPage(c, title)
	err := viewTpl.ExecuteWriter(pongo2.Context{"page": p}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func editHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	editTpl := pongo2.Must(pongo2.FromFile("view/edit.html"))
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
	http.Redirect(w, r, "/wiki/"+title, http.StatusFound)
}

func getWikiDb(c web.C) *WikiDb { return c.Env["wikidb"].(*WikiDb) }

func signupHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	signupTpl := pongo2.Must(pongo2.FromFile("view/signup.html"))
	err := signupTpl.ExecuteWriter(pongo2.Context{}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func HashPassword(password string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalln("Hash password failed: %v", err)
		panic(err)
	}
	return hash
}

func signupPostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	wikidb := getWikiDb(c)
	name := r.FormValue("username")
	password := r.FormValue("password")

	user := &User{Name: name, Password: HashPassword(password)}
	err := wikidb.DbMap.Insert(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session, _ := store.Get(r, SESSION_NAME)
	session.Values["id"] = user.Id
	sessions.Save(r, w)

	http.Redirect(w, r, "/wiki", http.StatusFound)
}

func loginHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	loginTpl := pongo2.Must(pongo2.FromFile("view/login.html"))
	err := loginTpl.ExecuteWriter(pongo2.Context{}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func loginPostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, SESSION_NAME)
	if session.IsNew {
		session.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7, // 1week
			HttpOnly: true}
	}

	delete(session.Values, "id")
	sessions.Save(r, w)

	wikidb := getWikiDb(c)
	name := r.FormValue("username")
	password := r.FormValue("password")

	user := User{}
	err := wikidb.DbMap.SelectOne(&user, "select * from user where name=?", name)
	if err == sql.ErrNoRows {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if err != nil {
		log.Fatalln(err)
	}

	err = bcrypt.CompareHashAndPassword(user.Password, []byte(password))
	if err != nil {
		// TODO: login failed
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	session.Values["id"] = user.Id
	sessions.Save(r, w)

	http.Redirect(w, r, "/wiki", http.StatusFound)
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

	return dbmap, err
}

func mainHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	mainTpl := pongo2.Must(pongo2.FromFile("view/main.html"))
	err := mainTpl.ExecuteWriter(pongo2.Context{}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func rootHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/wiki", http.StatusFound)
}

func logoutHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, SESSION_NAME)
	delete(session.Values, "id")
	sessions.Save(r, w)

	http.Redirect(w, r, "/wiki", http.StatusFound)
}

func markdownHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	tpl, err := pongo2.FromString("{{text|markdown|sanitize}}")
	if err != nil {
		panic(err)
	}

	r.ParseForm()
	text := r.FormValue("text")
	err = tpl.ExecuteWriter(pongo2.Context{"text": text}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func includeDb(dbmap *gorp.DbMap) func(c *web.C, h http.Handler) http.Handler {
	wikidb := &WikiDb{
		DbMap: dbmap,
	}

	return func(c *web.C, h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			c.Env["wikidb"] = wikidb
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func needLogin(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, SESSION_NAME)
		id, ok := session.Values["id"]
		if !ok {
			fmt.Printf("need login\n")
			http.Redirect(w, r, "/login", http.StatusFound)
		}

		wikidb := getWikiDb(*c)
		user := User{}
		err := wikidb.DbMap.SelectOne(&user, "select * from user where user_id=?", id)
		if err == sql.ErrNoRows {
			delete(session.Values, "id")
			sessions.Save(r, w)

			fmt.Printf("need login\n")
			http.Redirect(w, r, "/login", http.StatusFound)
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func addTestUser(dbmap *gorp.DbMap) {
	user := &User{
		Name: "test",
		Password: []byte("$2a$10$1KbzrHDRoPwZuHxWs1D6lOSLpcCRyPZXJ1Q7sPFbBf03DSc8y8n8K"),
	}

	dbmap.Insert(user)
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

	addTestUser(dbmap)

	m := web.New()
	m.Get("/signup", signupHandler)
	m.Get("/login", loginHandler)

	goji.Use(includeDb(dbmap))

	m.Post("/signup", signupPostHandler)
	m.Post("/login", loginPostHandler)

	m.Get("/logout", logoutHandler)
	m.Get("/wiki", mainHandler)
	m.Get("/", rootHandler)

	userMux := web.New()
	userMux.Use(needLogin)

	userMux.Use(includeDb(dbmap))

	userMux.Get("/wiki/:title", viewHandler)
	userMux.Get("/wiki/:title/edit", editHandler)
	userMux.Post("/wiki/:title", saveHandler)

	mdMux := web.New()
	mdMux.Use(needLogin)
	mdMux.Use(includeDb(dbmap))
	mdMux.Post("/markdown", markdownHandler)

	goji.Handle("/wiki/*", userMux)
	goji.Get("/assets/*", http.FileServer(http.Dir(".")))
	goji.Handle("/markdown", mdMux)
	goji.Handle("/*", m)

	goji.Serve()
}
