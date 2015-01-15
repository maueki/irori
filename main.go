package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"github.com/flosch/pongo2"
)

type Page struct {
	Title string
	Body []byte
}

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func viewHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	title := c.URLParams["title"]
	p, _ := loadPage(title)
	fmt.Fprintf(w, "<h1>%s</h1><div>%s</div>", p.Title, p.Body)
}

var editTpl = pongo2.Must(pongo2.FromFile("view/edit.html"))

func editHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	title := c.URLParams["title"]
	p, err := loadPage(title)
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
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/wiki/" + title, http.StatusFound)
}

func main() {
	goji.Get("/wiki/:title", viewHandler)
	goji.Get("/wiki/:title/edit", editHandler)
	goji.Post("/wiki/:title", saveHandler)
	goji.Serve()
}
