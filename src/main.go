package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/bkaradzic/go-lz4"
	"github.com/flosch/pongo2"
	_ "github.com/flosch/pongo2-addons"
	"github.com/gorilla/sessions"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var ErrUserNotFound = errors.New("user not found")

const SESSION_NAME = "irori_session"

var store = sessions.NewCookieStore([]byte("something-very-secret")) // FIXME

type AccessLevel string

const (
	PUBLIC  AccessLevel = "public"
	GROUP   AccessLevel = "group"
	PRIVATE AccessLevel = "private"
)

type page struct {
	Id       bson.ObjectId   `bson:"_id" json:"id"`
	Author   bson.ObjectId   `json:"author"`
	Article  article         `json:"article"`
	History  []history       `json:"-"`
	Projects []bson.ObjectId `json:"projects"`
	Access   AccessLevel     `json:"access"`
	Groups   []bson.ObjectId `json:"groups"`
}

type article struct {
	Id     bson.ObjectId `bson:"_id,omitempty" json:"id"`
	Title  string        `json:"title"`
	Body   string        `json:"body"`
	UserId bson.ObjectId `json:"userId"`
	Date   time.Time     `json:"date"`
}

type history struct {
	Id     bson.ObjectId `bson:"_id,omitempty"`
	Title  []byte
	Body   []byte
	UserId bson.ObjectId
	Date   time.Time
}

type permission string

const (
	ADMIN  permission = "admin"
	EDITOR permission = "editor"
)

type user struct {
	Id          bson.ObjectId          `bson:"_id,omitempty" json:"id"`
	Name        string                 `json:"name"`
	EMail       string                 `json:"email"`
	Password    []byte                 `json:"-"`
	Permissions map[permission]bool    `json:"permissions"`
	Projects    map[bson.ObjectId]bool `json:"projects"`
	Disabled    bool                   `json:"disabled"`
}

type project struct {
	Id       bson.ObjectId `bson:"_id,omitempty" json:"id,omitempty"`
	Name     string        `json:"name"`
	SlackURL string        `bson:"slackurl,omitempty" json:"slackurl,omitempty"`
}

func (u *user) HasPermission(perm permission) bool {
	b, ok := u.Permissions[perm]
	return ok && b
}

type docdb struct {
	Db *mgo.Database
}

func encodeFromText(text string) ([]byte, error) {
	return lz4.Encode(nil, []byte(text))
}

func decodeFromBlob(data []byte) (string, error) {
	data, error := lz4.Decode(nil, data)
	return string(data), error
}

func (a *article) createHistoryData() (*history, error) {
	title, err := encodeFromText(a.Title)
	if err != nil {
		return nil, err
	}

	body, err := encodeFromText(a.Body)
	if err != nil {
		return nil, err
	}

	history := history{}
	history.Id = a.Id
	history.Title = title
	history.Body = body
	history.UserId = a.UserId
	history.Date = a.Date

	return &history, nil
}

func (p *page) save(c web.C, r *http.Request) error {
	user := getSessionUser(c)

	p.Article.Id = bson.NewObjectId()
	history, err := p.Article.createHistoryData()
	if err != nil {
		return err
	}

	p.Article.UserId = user.Id
	p.Article.Date = time.Now()

	docdb := getDocDb(c)

	return docdb.Db.C("pages").UpdateId(p.Id,
		bson.M{"$set": bson.M{"article": p.Article, "projects": p.Projects, "access": p.Access, "groups": p.Groups},
			"$push": bson.M{"history": history}})
}

func getPageFromDb(c web.C, pageId string) (*page, error) {
	docdb := getDocDb(c)

	if !bson.IsObjectIdHex(pageId) {
		return nil, mgo.ErrNotFound
	}

	id := bson.ObjectIdHex(pageId)

	p := page{}
	err := docdb.Db.C("pages").FindId(id).One(&p)
	if err != nil {
		fmt.Printf("getPageFromDb failed : %s\n", pageId)
		return nil, err
	}

	fmt.Printf("getPageFromDb success : %s\n", pageId)

	return &p, nil
}

func getUserById(db *mgo.Database, id bson.ObjectId) (*user, error) {
	user := user{}
	err := db.C("users").FindId(id).One(&user)
	return &user, err
}

func executeWriterFromFile(w http.ResponseWriter, path string, context *pongo2.Context) error {
	tpl := pongo2.Must(pongo2.FromFile(path))
	return tpl.ExecuteWriter(*context, w)
}

// precond: must call after needLogin()
func getSessionUser(c web.C) *user {
	u, ok := c.Env["user"]
	if !ok {
		log.Fatalln("user not found")
	}

	retu, ok := u.(*user)
	if !ok {
		log.Fatalln("invalid user")
	}

	return retu
}

func createNewPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	user := getSessionUser(c)

	docdb := getDocDb(c)
	projects := []project{}

	err := docdb.Db.C("projects").Find(bson.M{}).All(&projects)
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
	docdb := getDocDb(c)
	editeduser, err := getUserById(docdb.Db, page.Article.UserId)
	if err == mgo.ErrNotFound {
		// TODO : when user is removed?
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// time.location
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	edittime := page.Article.Date.In(jst)

	// genarate html
	pongoCtx := pongo2.Context{
		"loginuser":  user,
		"pageid":     page.Id.Hex(),
		"edittime":   edittime.Format("2006/01/02 15:04"),
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

	docdb := getDocDb(c)
	projects := []project{}

	err = docdb.Db.C("projects").Find(bson.M{}).All(&projects)
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

func searchPageGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	q := r.FormValue("q")
	if q == "" {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	log.Println("query:", q)

	err := executeWriterFromFile(w, "view/search.html", &pongo2.Context{"query": q})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getDocDb(c web.C) *docdb { return c.Env["docdb"].(*docdb) }

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

	docdb := getDocDb(c)
	name := r.FormValue("username")
	password := r.FormValue("password")

	user := user{}
	err := docdb.Db.C("users").Find(bson.M{"name": name}).One(&user)
	if err == nil {
		err = bcrypt.CompareHashAndPassword(user.Password, []byte(password))
		if err == nil {
			session.Values["userid"] = user.Id.Hex()
			sessions.Save(r, w)
			http.Redirect(w, r, "/home", http.StatusSeeOther)
			return
		}
	}

	w.WriteHeader(http.StatusUnauthorized)
	executeWriterFromFile(w, "view/login.html", &pongo2.Context{"error": "Incorrect username or password."})
}

func rootHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/home", http.StatusMovedPermanently)
}

func logoutPostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, SESSION_NAME)
	delete(session.Values, "userid")
	sessions.Save(r, w)

	http.Redirect(w, r, "/home", http.StatusFound)
}

func markdownPostHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	tpl, err := pongo2.FromString("{{text|markdown|sanitize}}")
	if err != nil {
		panic(err)
	}

	r.ParseForm()
	text := r.FormValue("text")
	w.Header().Set("Content-Type", "text/html")
	err = tpl.ExecuteWriter(pongo2.Context{"text": text}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func staticPageHandler(path string) func(c web.C, w http.ResponseWriter, r *http.Request) {
	return func(c web.C, w http.ResponseWriter, r *http.Request) {
		err := executeWriterFromFile(w, path, &pongo2.Context{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func includeDb(db *mgo.Database) func(c *web.C, h http.Handler) http.Handler {
	docdb := &docdb{
		Db: db,
	}

	return func(c *web.C, h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			c.Env["docdb"] = docdb
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func getUserIfLoggedin(c web.C, r *http.Request) (*user, error) {
	session, _ := store.Get(r, SESSION_NAME)
	id, ok := session.Values["userid"]
	if !ok {
		return nil, ErrUserNotFound
	}

	docdb := getDocDb(c)
	user, err := getUserById(docdb.Db, bson.ObjectIdHex(id.(string)))
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
	guestHash, _ := bcrypt.GenerateFromPassword([]byte("guest"), bcrypt.DefaultCost)
	u := &user{
		Name:     "guest",
		Password: guestHash,
	}

	_, err := db.C("users").Upsert(bson.M{"name": u.Name},
		bson.M{"$setOnInsert": u})
	if err != nil {
		log.Fatalln(err)
	}

	adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	admin := &user{
		Name:        "admin",
		Password:    adminHash,
		Permissions: map[permission]bool{ADMIN: true, EDITOR: true},
	}

	_, err = db.C("users").Upsert(bson.M{"name": admin.Name},
		bson.M{"$setOnInsert": admin})
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
	m.Get("/", rootHandler)

	loginUserActionMux := web.New()
	loginUserActionMux.Use(needLogin)
	loginUserActionMux.Get("/action/createNewPage", createNewPageGetHandler)

	adminMux := web.New()
	adminMux.Use(needLogin)
	adminMux.Use(needAdmin)
	adminMux.Get("/admin/adduser", staticPageHandler("view/adduser.html"))
	adminMux.Get("/admin/projects", staticPageHandler("view/projects.html"))
	adminMux.Get("/admin/projects/:projectId", projectEditHandler)
	adminMux.Get("/admin/groups", staticPageHandler("view/groups.html"))
	adminMux.Get("/admin/groups/:groupId", groupEditHandler)
	adminMux.Get("/admin/users", staticPageHandler("view/users.html"))
	adminMux.Get("/admin", staticPageHandler("view/admin.html"))

	apiMux := web.New()
	apiMux.Use(needLogin)
	apiMux.Get("/api/projects", apiProjectListGetHandler)
	apiMux.Get("/api/projects/:projectId", apiProjectGetHandler)
	apiMux.Put("/api/projects/:projectId", applyFilter(apiProjectPutHandler, apiNeedPermission(ADMIN)))
	apiMux.Post("/api/projects", applyFilter(apiProjectsPostHandler, apiNeedPermission(ADMIN)))

	apiMux.Get("/api/pages/own", apiOwnPageGetHandler)
	apiMux.Get("/api/pages/:pageId/body", apiPageBodyGetHandler)
	apiMux.Get("/api/pages/:pageId", apiPageGetHandler)
	apiMux.Get("/api/pages", apiPageListGetHandler)
	apiMux.Post("/api/pages/:pageId", applyFilter(apiPageUpdateHandler, apiNeedPermission(EDITOR)))
	apiMux.Post("/api/pages", applyFilter(apiPageCreateHandler, apiNeedPermission(EDITOR)))

	apiMux.Post("/api/groups", applyFilter(apiGroupCreateHandler, apiNeedPermission(ADMIN)))
	apiMux.Get("/api/groups/:groupId", apiGroupGetHandler)
	apiMux.Put("/api/groups/:groupId", applyFilter(apiGroupPutHandler, apiNeedPermission(ADMIN)))
	apiMux.Get("/api/groups", apiGroupListGetHandler)

	apiMux.Get("/api/users", apiUserListGetHandler)
	apiMux.Post("/api/users", applyFilter(apiUserPostHandler, apiNeedPermission(ADMIN)))
	apiMux.Get("/api/users/own", apiOwnUserGetHandler)
	apiMux.Get("/api/users/icon", apiOwnIconHandler)
	apiMux.Get("/api/users/:userId/icon", apiUserIconHandler)
	apiMux.Delete("/api/users/:userId", applyFilter(apiUserDeleteHandler, apiNeedPermission(ADMIN)))
	apiMux.Get("/api/users/:userId", apiUserGetHandler)

	apiMux.Put("/api/password", apiPasswordHandler)

	// Mux : create new page or show a page created already
	pageMux := web.New()
	pageMux.Use(needLogin)
	pageMux.Get("/docs", searchPageGetHandler)
	pageMux.Get("/docs/:pageId", viewPageGetHandler)
	pageMux.Get("/docs/:pageId/edit", editPageGetHandler)

	// Mux : convert Markdown to HTML which is send by Ajax
	mdMux := web.New()
	mdMux.Use(needLogin)
	mdMux.Post("/markdown", markdownPostHandler)

	homeMux := web.New()
	homeMux.Use(needLogin)
	homeMux.Get("/home", staticPageHandler("view/home-pages.html"))

	projectMux := web.New()
	projectMux.Use(needLogin)
	projectMux.Get("/project/:projectId", staticPageHandler("view/project.html"))

	profileMux := web.New()
	profileMux.Use(needLogin)
	profileMux.Get("/profile", staticPageHandler("view/profile.html"))
	profileMux.Get("/profile/password/edit", staticPageHandler("view/profile-password.html"))

	goji.Use(includeDb(db))
	goji.Get("/assets/*", http.FileServer(http.Dir(".")))
	goji.Handle("/docs/*", pageMux)
	goji.Handle("/docs", pageMux)
	goji.Handle("/home", homeMux)
	goji.Handle("/project/*", projectMux)
	goji.Handle("/markdown", mdMux)
	goji.Handle("/action/*", loginUserActionMux)
	goji.Handle("/admin/*", adminMux)
	goji.Handle("/admin", adminMux)
	goji.Handle("/api/*", apiMux)
	goji.Handle("/profile", profileMux)
	goji.Handle("/profile/*", profileMux)
	goji.Handle("/*", m)
}

type handleFilter func(web.C, http.ResponseWriter, *http.Request) bool

func applyFilter_(h func(web.C, http.ResponseWriter, *http.Request), fs []handleFilter) func(web.C, http.ResponseWriter, *http.Request) {
	if len(fs) == 0 {
		return h
	}

	newhandler := func(c web.C, w http.ResponseWriter, r *http.Request) {
		if fs[0](c, w, r) {
			h(c, w, r)
		}
	}

	return applyFilter_(newhandler, fs[1:])
}

func applyFilter(h func(web.C, http.ResponseWriter, *http.Request), fs ...handleFilter) func(web.C, http.ResponseWriter, *http.Request) {
	return applyFilter_(h, fs)
}

func apiNeedPermission(p permission) handleFilter {
	return func(c web.C, w http.ResponseWriter, r *http.Request) bool {
		user := getSessionUser(c)

		if !user.HasPermission(p) {
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}

		return true
	}
}

type iroriconfig struct {
	HostName string
	Port     int
}

var IroriConfig iroriconfig

func Initialize() {

	AddDecoder(&IroriConfig)
	ReadConfig()

	hostname := os.Getenv("IRORI_HOSTNAME")
	if hostname != "" {
		IroriConfig.HostName = hostname
	}
	if IroriConfig.HostName == "" {
		IroriConfig.HostName = "localhost"
	}

	port, err := strconv.Atoi(os.Getenv("IRORI_PORT"))
	if err != nil && port != 0 {
		IroriConfig.Port = port
	}

	log.Println(IroriConfig)
}

func main() {
	Initialize()

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

	pageHooks = append(pageHooks, pageHookSlack{db: db})

	setRoute(db)

	goji.Serve()
}
