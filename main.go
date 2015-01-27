package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"code.google.com/p/go.crypto/bcrypt"

	"github.com/coopernurse/gorp"
	"github.com/flosch/pongo2"
	_ "github.com/flosch/pongo2-addons"
	_ "github.com/mattn/go-sqlite3"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"github.com/bkaradzic/go-lz4"
	"github.com/gorilla/sessions"
)

const SESSION_NAME = "go_wiki_session"

var store = sessions.NewCookieStore([]byte("something-very-secret")) // FIXME

type Page struct {
	Id                 int64 `db:"post_id"`
	Title              string
	Body               string
	LastModifiedUserId int64
	LastModifiedDate   time.Time
}

type History struct {
	Id             int64 `db:"history_id"`
	PageId         int64
	Title          []byte
	Body           []byte
	ModifiedUserId int64
	ModifiedDate   time.Time
}

type User struct {
	Id       int64  `db:"user_id"`
	Name     string `db:"name"`
	Password []byte `db:"password"`
}

type WikiDb struct {
	DbMap *gorp.DbMap
}

type LoginUser struct {
	Exist bool
	Name  string
}

type TransactionChain struct {
	Transaction *gorp.Transaction
	Error       error
}

func createTransaction(DbMap *gorp.DbMap) (*TransactionChain, error) {
	t := TransactionChain{}
	trans, err := DbMap.Begin()

	if err != nil {
		return nil, err
	}

	t.Transaction = trans
	return &t, nil
}

func (t *TransactionChain) Insert(list ...interface{}) *TransactionChain {
	if t.Error == nil {
		t.Error = t.Transaction.Insert(list...)
	}

	return t
}

func (t *TransactionChain) Update(list ...interface{}) *TransactionChain {
	if t.Error == nil {
		_, t.Error = t.Transaction.Update(list...)
	}

	return t
}

func (t *TransactionChain) Subscribe() error {
	if t.Error != nil {
		t.Transaction.Rollback()
		return t.Error
	}

	return t.Transaction.Commit()
}

func getUserId(r *http.Request) (int64, bool) {
	session, _ := store.Get(r, SESSION_NAME)

	if id, ok := session.Values["id"].(int64); ok {
		return id, true
	} else {
		return 0, false
	}
}

func encodeFromText(text string) ([]byte, error) {
	return lz4.Encode(nil, []byte(text))
}

func decodeFromBlob(data []byte) (string, error) {
	data, error := lz4.Decode(nil, data)
	return string(data), error
}

func (p *Page) createHistoryData() (*History, error) {
	title, err := encodeFromText(p.Title)
	if err != nil {
		return nil, err
	}

	body, err := encodeFromText(p.Body)
	if err != nil {
		return nil, err
	}

	history := History{}
	history.Title = title
	history.Body = body
	history.ModifiedUserId = p.LastModifiedUserId
	history.ModifiedDate = p.LastModifiedDate

	return &history, nil
}

func (p *Page) save(c web.C, r *http.Request) error {
	id, hasid := getUserId(r)
	if !hasid {
		return errors.New("failed to get user id.") // FIXME
	}

	p.LastModifiedUserId = id
	p.LastModifiedDate = time.Now()

	history, err := p.createHistoryData()
	if err != nil {
		return err
	}

	wikidb := getWikiDb(c)
	pOld := Page{}

	err = wikidb.DbMap.SelectOne(&pOld, "select * from page where title=?", p.Title)
	if err == sql.ErrNoRows {
		t, err := createTransaction(wikidb.DbMap)
		if err != nil {
			return err
		}

		t.Insert(p)

		history.PageId = p.Id
		t.Insert(history)

		return t.Subscribe()
	} else if err != nil {
		log.Fatalln(err)
	}

	p.Id = pOld.Id
	history.PageId = p.Id

	t, err := createTransaction(wikidb.DbMap)
	t.Update(p).Insert(history)
	return t.Subscribe()
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

func getLoginUserInfo(c web.C, w http.ResponseWriter, r *http.Request) (*LoginUser, error) {
	// check current user is success to login or not
	loginuser := LoginUser{false, ""}

	session, _ := store.Get(r, SESSION_NAME)
	id, ok := session.Values["id"]
	if ok {
		wikidb := getWikiDb(c)
		user := User{}
		err := wikidb.DbMap.SelectOne(&user, "select * from user where user_id=?", id)
		if err == sql.ErrNoRows {
			delete(session.Values, "id")
			sessions.Save(r, w)
			fmt.Printf("not login\n")
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			loginuser.Exist = true
			loginuser.Name = user.Name
		}
	}

	return &loginuser, nil
}

func viewHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	viewTpl := pongo2.Must(pongo2.FromFile("view/view.html"))
	title := c.URLParams["title"]
	p, _ := loadPage(c, title)
	loginuser, _ := getLoginUserInfo(c, w, r)
	err := viewTpl.ExecuteWriter(pongo2.Context{"loginuser": loginuser, "page": p}, w)
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

	loginuser, _ := getLoginUserInfo(c, w, r)
	err = editTpl.ExecuteWriter(pongo2.Context{"loginuser": loginuser, "page": p}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func saveHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	title := c.URLParams["title"]
	body := r.FormValue("body")
	p := &Page{Title: title, Body: body}
	err := p.save(c, r)
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
	pageTable.ColMap("LastModifiedUserId").Rename("lastuser")
	pageTable.ColMap("LastModifiedDate").Rename("lastdate")

	historyTable := dbmap.AddTableWithName(History{}, "history").SetKeys(true, "Id")
	historyTable.ColMap("PageId").Rename("pageid")
	historyTable.ColMap("Title").Rename("title")
	historyTable.ColMap("Body").Rename("body")
	historyTable.ColMap("ModifiedUserId").Rename("user")
	historyTable.ColMap("ModifiedDate").Rename("date")

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

	loginuser, _ := getLoginUserInfo(c, w, r)
	err := mainTpl.ExecuteWriter(pongo2.Context{"loginuser": loginuser}, w)
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
		Name:     "test",
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
