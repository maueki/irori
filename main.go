package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"code.google.com/p/go.crypto/bcrypt"
	"github.com/bkaradzic/go-lz4"
	"github.com/flosch/pongo2"
	_ "github.com/flosch/pongo2-addons"
	"github.com/gorilla/sessions"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var ErrUserNotFound = errors.New("user not found")

const SESSION_NAME = "go_wiki_session"

var store = sessions.NewCookieStore([]byte("something-very-secret")) // FIXME

type AccessLevel string

const (
	PUBLIC  AccessLevel = "public"
	GROUP   AccessLevel = "group"
	PRIVATE AccessLevel = "private"
)

type Page struct {
	Id       bson.ObjectId `bson:"_id"`
	Article  Article
	History  []History `json:"-"`
	Projects []bson.ObjectId
	Access   AccessLevel
	Groups   []bson.ObjectId
}

type Article struct {
	Id     bson.ObjectId `bson:"_id,omitempty"`
	Title  string
	Body   string
	UserId bson.ObjectId
	Date   time.Time
}

type History struct {
	Id     bson.ObjectId `bson:"_id,omitempty"`
	Title  []byte
	Body   []byte
	UserId bson.ObjectId
	Date   time.Time
}

type Permission string

const (
	ADMIN  Permission = "admin"
	EDITOR Permission = "editor"
)

type User struct {
	Id          bson.ObjectId `bson:"_id,omitempty"`
	Name        string
	Password    []byte
	Permissions map[Permission]bool
	Projects    map[bson.ObjectId]bool
}

type Project struct {
	Id   bson.ObjectId `bson:"_id,omitempty" json:"id,omitempty"`
	Name string
}

func (u *User) HasPermission(perm Permission) bool {
	b, ok := u.Permissions[perm]
	return ok && b
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
	history.Id = a.Id
	history.Title = title
	history.Body = body
	history.UserId = a.UserId
	history.Date = a.Date

	return &history, nil
}

func (p *Page) save(c web.C, r *http.Request) error {
	user := getSessionUser(c)

	p.Article.Id = bson.NewObjectId()
	history, err := p.Article.createHistoryData()
	if err != nil {
		return err
	}

	p.Article.UserId = user.Id
	p.Article.Date = time.Now()

	wikidb := getWikiDb(c)

	return wikidb.Db.C("pages").UpdateId(p.Id,
		bson.M{"$set": bson.M{"article": p.Article, "projects": p.Projects},
			"$push": bson.M{"history": history}})
}

func getPageFromDb(c web.C, pageId string) (*Page, error) {
	wikidb := getWikiDb(c)

	if !bson.IsObjectIdHex(pageId) {
		return nil, mgo.ErrNotFound
	}

	id := bson.ObjectIdHex(pageId)

	p := Page{}
	err := wikidb.Db.C("pages").FindId(id).One(&p)
	if err != nil {
		fmt.Printf("getPageFromDb failed : %s\n", pageId)
		return nil, err
	}

	fmt.Printf("getPageFromDb success : %s\n", pageId)

	return &p, nil
}

func getUserById(db *mgo.Database, id bson.ObjectId) (*User, error) {
	user := User{}
	err := db.C("users").FindId(id).One(&user)
	return &user, err
}

func executeWriterFromFile(w http.ResponseWriter, path string, context *pongo2.Context) error {
	tpl := pongo2.Must(pongo2.FromFile(path))
	return tpl.ExecuteWriter(*context, w)
}

// precond: must call after needLogin()
func getSessionUser(c web.C) *User {
	user, ok := c.Env["user"]
	if !ok {
		log.Fatalln("user not found")
	}

	u, ok := user.(*User)
	if !ok {
		log.Fatalln("invalid user")
	}

	return u
}

func createNewPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	user := getSessionUser(c)

	wikidb := getWikiDb(c)
	projects := []Project{}

	err := wikidb.Db.C("projects").Find(bson.M{}).All(&projects)
	if err != nil {
		log.Fatal("@@@ projects")
	}

	err = executeWriterFromFile(w, "view/newpage.html",
		&pongo2.Context{
			"isEditor": user.HasPermission(EDITOR),
			"projects": projects,
		})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	pageId := c.URLParams["pageId"]

	// get page info
	page, err := getPageFromDb(c, pageId)
	if page == nil || err != nil {
		// FIXME : redirect to top page or "NotFound" page
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// get current login user info
	user := getSessionUser(c)

	// get last edited user info
	wikidb := getWikiDb(c)
	editeduser, err := getUserById(wikidb.Db, page.Article.UserId)
	if err == mgo.ErrNotFound {
		// TODO : when user is removed?
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// genarate html
	pongoCtx := pongo2.Context{
		"loginuser":  user,
		"page":       page,
		"editeduser": editeduser}

	err = executeWriterFromFile(w, "view/view.html", &pongoCtx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func editPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	pageId := c.URLParams["pageId"]

	page, err := getPageFromDb(c, pageId)
	if err != nil {
		// FIXME : redirect to top page or "NotFound" page
		http.Error(w, err.Error(), http.StatusNotFound)
	}

	user := getSessionUser(c)

	wikidb := getWikiDb(c)
	projects := []Project{}

	err = wikidb.Db.C("projects").Find(bson.M{}).All(&projects)
	if err != nil {
		log.Fatal("@@@ projects")
	}

	log.Println("Projects:", projects)
	log.Println("page Projects:", page.Projects)

	err = executeWriterFromFile(w, "view/edit.html",
		&pongo2.Context{
			"loginuser": user,
			"page":      page,
			"isEditor":  user.HasPermission(EDITOR),
			"projects":  projects,
		})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func savePagePostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	user := getSessionUser(c)
	if !user.HasPermission(EDITOR) {
		http.Error(w, "You are not editor", http.StatusMethodNotAllowed)
		return
	}

	pageId := c.URLParams["pageId"]

	p, err := getPageFromDb(c, pageId)
	if err != nil {
		// FIXME : redirect to top page or "NotFound" page
		http.Error(w, err.Error(), http.StatusNotFound)
	}

	p.Article.Title = r.FormValue("title")
	p.Article.Body = r.FormValue("body")

	pids := []bson.ObjectId{}
	for _, v := range r.Form["projects"] {
		// FIXME: check project id
		if pid := bson.ObjectIdHex(v); pid.Valid() {
			pids = append(pids, pid)
		}
	}
	p.Projects = pids

	err = p.save(c, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/wiki/"+pageId, http.StatusFound)
}

func getWikiDb(c web.C) *WikiDb { return c.Env["wikidb"].(*WikiDb) }

func HashPassword(password string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalln("Hash password failed: %v", err)
		panic(err)
	}
	return hash
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
	err := wikidb.Db.C("users").Find(bson.M{"name": name}).One(&user)
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

func topPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	user, err := getUserIfLoggedin(c, r)
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

func addUserGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	err := executeWriterFromFile(w, "view/adduser.html", &pongo2.Context{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func addUserPostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	wikidb := getWikiDb(c)
	name := r.FormValue("username")
	password := r.FormValue("password")

	user := &User{
		Name:        name,
		Password:    HashPassword(password),
		Permissions: map[Permission]bool{EDITOR: true},
	}

	// Register user only if not found.
	changeinfo, err := wikidb.Db.C("users").Upsert(bson.M{"name": name},
		bson.M{"$setOnInsert": user})
	if err != nil {
		log.Println(err)
		executeWriterFromFile(w, "view/adduser.html", &pongo2.Context{"error": "Incorrect, please try again."})
		return
	}

	if changeinfo.UpsertedId == nil {
		log.Println("user.Name already exists:", name)
		executeWriterFromFile(w, "view/adduser.html", &pongo2.Context{"error": "Incorrect, please try again."})
		return
	}

	http.Redirect(w, r, "/wiki", http.StatusFound)
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

func getUserIfLoggedin(c web.C, r *http.Request) (*User, error) {
	session, _ := store.Get(r, SESSION_NAME)
	id, ok := session.Values["userid"]
	if !ok {
		return nil, ErrUserNotFound
	}

	wikidb := getWikiDb(c)
	user, err := getUserById(wikidb.Db, bson.ObjectIdHex(id.(string)))
	if err == mgo.ErrNotFound {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, err
	}

	return user, err
}

func needLogin(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user, err := getUserIfLoggedin(*c, r)

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

func needAdmin(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user := getSessionUser(*c)

		if !user.HasPermission(ADMIN) {
			http.Error(w, "You are not admin", http.StatusMethodNotAllowed)
			return
		}

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func addTestData(db *mgo.Database) {
	db.C("users").RemoveAll(nil)    // FIXME
	db.C("projects").RemoveAll(nil) // FIXME
	db.C("groups").RemoveAll(nil)   // FIXME

	guestHash, _ := bcrypt.GenerateFromPassword([]byte("guest"), bcrypt.DefaultCost)
	user := &User{
		Name:     "guest",
		Password: guestHash,
	}

	err := db.C("users").Insert(user)
	if err != nil {
		log.Fatalln(err)
	}

	admin := &User{
		Name:        "admin",
		Password:    []byte("$2a$10$yEuWec8ND/E6CoX3jsbfpu9nXX7PNH7ki6hwyb9RvqNm6ZPdjakCm"),
		Permissions: map[Permission]bool{ADMIN: true, EDITOR: true},
	}

	err = db.C("users").Insert(admin)
	if err != nil {
		log.Fatalln(err)
	}

	projIrori := &Project{
		Name: "irori"}

	err = db.C("projects").Insert(projIrori)
	if err != nil {
		log.Fatalln(err)
	}
}

func setRoute(db *mgo.Database) {
	addTestData(db)

	m := web.New()
	m.Get("/login", loginPageGetHandler)
	m.Post("/login", loginPostHandler)
	m.Post("/logout", logoutPostHandler)
	m.Get("/wiki", topPageGetHandler)
	m.Get("/", rootHandler)

	loginUserActionMux := web.New()
	loginUserActionMux.Use(needLogin)
	loginUserActionMux.Get("/action/createNewPage", createNewPageGetHandler)

	adminMux := web.New()
	adminMux.Use(needLogin)
	adminMux.Use(needAdmin)
	adminMux.Get("/admin/adduser", addUserGetHandler)
	adminMux.Post("/admin/adduser", addUserPostHandler)
	adminMux.Get("/admin/projects", projectsGetHandler)
	adminMux.Get("/admin/groups", groupsGetHandler)
	adminMux.Get("/admin/groups/:groupId", groupEditHandler)

	apiMux := web.New()
	apiMux.Use(needLogin)
	apiMux.Get("/api/projects", apiProjectsGetHandler)
	apiMux.Post("/api/projects", apiProjectsPostHandler)

	apiMux.Get("/api/pages/:pageId", apiPageGetHandler)
	apiMux.Post("/api/pages/:pageId", apiPageUpdateHandler)
	apiMux.Post("/api/pages", apiPageCreateHandler)

	apiMux.Post("/api/groups", apiGroupCreateHandler)
	apiMux.Get("/api/groups/:groupId", apiGroupGetHandler)
	apiMux.Put("/api/groups/:groupId", apiGroupPutHandler)
	apiMux.Get("/api/groups", apiGroupListGetHandler)

	apiMux.Get("/api/users", apiUserListGetHandler)

	// Mux : create new page or show a page created already
	pageMux := web.New()
	pageMux.Use(needLogin)
	pageMux.Get("/wiki/:pageId", viewPageGetHandler)
	pageMux.Get("/wiki/:pageId/edit", editPageGetHandler)
	pageMux.Post("/wiki/:pageId", savePagePostHandler)

	// Mux : convert Markdown to HTML which is send by Ajax
	mdMux := web.New()
	mdMux.Use(needLogin)
	mdMux.Post("/markdown", markdownPostHandler)

	goji.Use(includeDb(db))
	goji.Get("/assets/*", http.FileServer(http.Dir(".")))
	goji.Handle("/wiki/*", pageMux)
	goji.Handle("/markdown", mdMux)
	goji.Handle("/action/*", loginUserActionMux)
	goji.Handle("/admin/*", adminMux)
	goji.Handle("/api/*", apiMux)
	goji.Handle("/*", m)
}

func main() {
	ReadConfig()

	url := os.Getenv("MONGODB_URL")
	if url == "" {
		url = "localhost/irori"
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
