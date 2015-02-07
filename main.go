package main

import (
	"database/sql"
	"errors"
	"fmt"
	//"github.com/maueki/go_wiki/db"
	"log"
	"net/http"
	"os"
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

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var ErrUserNotFound = errors.New("user not found")

const SESSION_NAME = "go_wiki_session"

var store = sessions.NewCookieStore([]byte("something-very-secret")) // FIXME

type Page struct {
	Id      bson.ObjectId `bson:"_id"`
	Article Article
	History []History
}

type UserRef struct {
	Ref string        `bson:"$ref"`
	Id  bson.ObjectId `bson:"$id"`
}

type Article struct {
	Title string
	Body  string
	User  UserRef
	Date  time.Time
}

type History struct {
	Title []byte
	Body  []byte
	User  UserRef
	Date  time.Time
}

type User struct {
	Id       bson.ObjectId `bson:"_id,omitempty"`
	Name     string
	Password []byte
}

type WikiDb struct {
	Db *mgo.Database
}

func encodeFromText(text string) ([]byte, error) {
	return lz4.Encode(nil, []byte(text))
}

func decodeFromBlob(data []byte) (string, error) {
	data, error := lz4.Decode(nil, data)
	return string(data), error
}

func (a *Article) createHistoryData() (*History, error) {
	title, err := encodeFromText(a.Title)
	if err != nil {
		return nil, err
	}

	body, err := encodeFromText(a.Body)
	if err != nil {
		return nil, err
	}

	history := History{}
	history.Title = title
	history.Body = body
	history.User = a.User
	history.Date = a.Date

	return &history, nil
}

func (p *Page) save(c web.C, r *http.Request) error {
	user, ok := getUser(c)
	if !ok {
		return errors.New("failed to get user id.") // FIXME
	}

	history, err := p.Article.createHistoryData()
	if err != nil {
		return err
	}

	p.Article.User.Id = user.Id
	p.Article.User.Ref = "user"
	p.Article.Date = time.Now()

	wikidb := getWikiDb(c)
	return wikidb.Db.C("page").UpdateId(p.Id,
		bson.M{"$set": bson.M{"article": p.Article},
			"$push": bson.M{"history": history}})
}

func getPageFromDb(c web.C, pageId string) (*Page, error) {
	wikidb := getWikiDb(c)

	if !bson.IsObjectIdHex(pageId) {
		return nil, mgo.ErrNotFound
	}

	id := bson.ObjectIdHex(pageId)

	p := Page{}
	err := wikidb.Db.C("page").FindId(id).One(&p)
	if err != nil {
		fmt.Printf("getPageFromDb failed : %s\n", pageId)
		return nil, err
	}

	fmt.Printf("getPageFromDb success : %s\n", pageId)

	return &p, nil
}

func getUserById(db *mgo.Database, id bson.ObjectId) (*User, error) {
	user := User{}
	err := db.C("user").FindId(id).One(&user)
	return &user, err
}

func executeWriterFromFile(w http.ResponseWriter, path string, context *pongo2.Context) error {
	tpl := pongo2.Must(pongo2.FromFile(path))
	return tpl.ExecuteWriter(*context, w)
}

// precond: must call after needLogin()
func getUser(c web.C) (*User, bool) {
	user, ok := c.Env["user"]
	if !ok {
		return nil, false
	}

	u, ok := user.(*User)
	return u, ok
}

func createNewPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	wikidb := getWikiDb(c)

	user, ok := getUser(c)
	if !ok {
		http.Error(w, "user not logined", http.StatusInternalServerError)
		return
	}

	p := &Page{
		Id: bson.NewObjectId(),
		Article: Article{Title: "", Body: "", Date: time.Now(), User: UserRef{Id: user.Id, Ref: "user"}}}

	fmt.Println(p.Id.Hex())
	err := wikidb.Db.C("page").Insert(p)
	if err != nil {
		log.Fatalln("createNewPage:", err)
	}
	pageId := p.Id.Hex()

	fmt.Printf("createNewPage pageId_str : %s\n", pageId)

	http.Redirect(w, r, "/wiki/"+pageId+"/edit", http.StatusFound)
}

func viewPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	pageId := c.URLParams["pageId"]

	p, err := getPageFromDb(c, pageId)
	if p == nil || err != nil {
		// FIXME : redirect to top page or "NotFound" page
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	user, ok := getUser(c)
	if !ok {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = executeWriterFromFile(w, "view/view.html",
		&pongo2.Context{"loginuser": user, "page": p})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func editPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	pageId := c.URLParams["pageId"]

	p, err := getPageFromDb(c, pageId)
	if err != nil {
		// FIXME : redirect to top page or "NotFound" page
		http.Error(w, err.Error(), http.StatusNotFound)
	}

	user, ok := getUser(c)
	if !ok {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = executeWriterFromFile(w, "view/edit.html",
		&pongo2.Context{"loginuser": user, "page": p})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func savePagePostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	pageId := c.URLParams["pageId"]

	fmt.Println("pageid:", pageId)

	p, err := getPageFromDb(c, pageId)
	if err != nil {
		// FIXME : redirect to top page or "NotFound" page
		http.Error(w, err.Error(), http.StatusNotFound)
	}

	p.Article.Title = r.FormValue("title")
	p.Article.Body = r.FormValue("body")
	err = p.save(c, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/wiki/"+pageId, http.StatusFound)
}

func getWikiDb(c web.C) *WikiDb { return c.Env["wikidb"].(*WikiDb) }

func signupPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	err := executeWriterFromFile(w, "view/signup.html", &pongo2.Context{})
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

	// Register user only if not found.
	changeinfo, err := wikidb.Db.C("user").Upsert(bson.M{"name": name},
		bson.M{"$setOnInsert": user})
	if err != nil {
		log.Println(err)
		executeWriterFromFile(w, "view/signup.html", &pongo2.Context{"error": "Incorrect, please try again."})
		return
	}

	if changeinfo.UpsertedId == nil {
		log.Println("user.Name already exists:", name)
		executeWriterFromFile(w, "view/signup.html", &pongo2.Context{"error": "Incorrect, please try again."})
		return
	}

	session, _ := store.Get(r, SESSION_NAME)
	session.Values["userid"] = changeinfo.UpsertedId.(bson.ObjectId).Hex()
	sessions.Save(r, w)

	http.Redirect(w, r, "/wiki", http.StatusFound)
}

func loginPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	err := executeWriterFromFile(w, "view/login.html", &pongo2.Context{})
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

	delete(session.Values, "userid")
	sessions.Save(r, w)

	wikidb := getWikiDb(c)
	name := r.FormValue("username")
	password := r.FormValue("password")

	user := User{}
	err := wikidb.Db.C("user").Find(bson.M{"name": name}).One(&user)
	if err == nil {
		err = bcrypt.CompareHashAndPassword(user.Password, []byte(password))
		if err == nil {
			session.Values["userid"] = user.Id.Hex()
			sessions.Save(r, w)
			http.Redirect(w, r, "/wiki", http.StatusSeeOther)
			return
		}
	}

	w.WriteHeader(http.StatusUnauthorized)
	executeWriterFromFile(w, "view/login.html", &pongo2.Context{"error": "Incorrect username or password."})
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

	dbmap.DropTables()
	err := dbmap.CreateTables()
	if err != nil {
		return nil, err
	}

	return dbmap, err
}

func topPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromDB(c, r)
	if err != nil {
		if err == ErrUserNotFound {
			err := executeWriterFromFile(w, "view/prelogin.html", &pongo2.Context{})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		err := executeWriterFromFile(w, "view/main.html", &pongo2.Context{"loginuser": user})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func rootHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/wiki", http.StatusMovedPermanently)
}

func logoutPostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, SESSION_NAME)
	delete(session.Values, "userid")
	sessions.Save(r, w)

	http.Redirect(w, r, "/wiki", http.StatusFound)
}

func markdownPostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
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

func includeDb(db *mgo.Database) func(c *web.C, h http.Handler) http.Handler {
	wikidb := &WikiDb{
		Db: db,
	}

	return func(c *web.C, h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			c.Env["wikidb"] = wikidb
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func getUserFromDB(c web.C, r *http.Request) (*User, error) {
	session, _ := store.Get(r, SESSION_NAME)
	id, ok := session.Values["userid"]
	if !ok {
		return nil, ErrUserNotFound
	}

	wikidb := getWikiDb(c)
	user, err := getUserById(wikidb.Db, bson.ObjectIdHex(id.(string)))
	return user, err
}

func needLogin(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user, err := getUserFromDB(*c, r)

		if err == ErrUserNotFound {
			session, _ := store.Get(r, SESSION_NAME)
			delete(session.Values, "userid")
			sessions.Save(r, w)

			http.Redirect(w, r, "/login", http.StatusFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			c.Env["user"] = user
		}

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func addTestUser(db *mgo.Database) {
	db.C("user").RemoveAll(nil) // FIXME

	user := &User{
		Name: "test",
		Password: []byte("$2a$10$1KbzrHDRoPwZuHxWs1D6lOSLpcCRyPZXJ1Q7sPFbBf03DSc8y8n8K"),
	}

	err := db.C("user").Insert(user)
	if err != nil {
		log.Fatalln(err)
	}
}

func setRoute(db *mgo.Database) {
	addTestUser(db)

	m := web.New()
	m.Get("/signup", signupPageGetHandler)
	m.Get("/login", loginPageGetHandler)
	m.Post("/signup", signupPostHandler)
	m.Post("/login", loginPostHandler)
	m.Post("/logout", logoutPostHandler)
	m.Get("/wiki", topPageGetHandler)
	m.Get("/", rootHandler)

	loginUserActionMux := web.New()
	loginUserActionMux.Use(needLogin)
	loginUserActionMux.Use(includeDb(db))
	loginUserActionMux.Get("/action/createNewPage", createNewPageGetHandler)

	// Mux : create new page or show a page created already
	pageMux := web.New()
	pageMux.Use(needLogin)
	pageMux.Use(includeDb(db))
	pageMux.Get("/wiki/:pageId", viewPageGetHandler)
	pageMux.Get("/wiki/:pageId/edit", editPageGetHandler)
	pageMux.Post("/wiki/:pageId", savePagePostHandler)

	// Mux : convert Markdown to HTML which is send by Ajax
	mdMux := web.New()
	mdMux.Use(needLogin)
	mdMux.Use(includeDb(db))
	mdMux.Post("/markdown", markdownPostHandler)

	goji.Use(includeDb(db))
	goji.Get("/assets/*", http.FileServer(http.Dir(".")))
	goji.Handle("/wiki/*", pageMux)
	goji.Handle("/markdown", mdMux)
	goji.Handle("/action/*", loginUserActionMux)
	goji.Handle("/*", m)
}

func main() {
	ReadConfig()

	url := os.Getenv("MONGODB_URL")
	if url == "" {
		url = "localhost/gowiki"
	}

	session, err := mgo.Dial(url)
	if err != nil {
		log.Fatalln(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)

	db := session.DB("")

	setRoute(db)

	goji.Serve()
}
