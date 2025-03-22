package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Page struct {
	Title string
	Body  []byte
}

const data_folder string = "data"

func (p *Page) save(title string) error {
	if title != p.Title && title != "new" {
		tmpRemove := title + ".txt"
		err := os.Remove(filepath.Join(data_folder, tmpRemove))
		if err != nil {
			return err
		}
	}
	filename := p.Title + ".txt"
	filePath := filepath.Join(data_folder, filename)
	return os.WriteFile(filePath, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	filePath := filepath.Join(data_folder, filename)
	body, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, temp string, p *Page) {
	filename := temp + ".html"
	t, err := template.ParseFiles(filepath.Join("templates", filename))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, p)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/view/"):]
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/edit/"):]
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/save/"):]
	name := r.FormValue("name")
	body := r.FormValue("body")
	p := &Page{Title: name, Body: []byte(body)}
	err := p.save(title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+name, http.StatusFound)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	pages, err := os.ReadDir("data")
	if err != nil {
		http.Error(w, "chyba pri citani directory data", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<form action=\"edit/new\">"+"<input type=\"submit\" value=\"Create note\"\\>"+"</form>")
	fmt.Fprintf(w, "<ul>")
	for _, page := range pages {
		name := page.Name()
		name = strings.ReplaceAll(name, ".txt", "")
		fmt.Fprintf(w, "<li><a href=\"/view/%s\">%s</a></li>", name, name)
	}
	fmt.Fprintf(w, "</ul>")
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/delete/"):]
	filename := title + ".txt"
	err := os.Remove(filepath.Join("data", filename))
	if err != nil {
		http.Error(w, "chyba pri mazani suboru", http.StatusInternalServerError)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func main() {
	err := os.MkdirAll(data_folder, os.ModePerm)
	if err != nil {
		log.Fatal("nedokazal som vytvorit dir", err)
	}
	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/edit/", editHandler)
	http.HandleFunc("/save/", saveHandler)
	http.HandleFunc("/detele/", deleteHandler)
	http.HandleFunc("/", listHandler)
	log.Fatal(http.ListenAndServe(":42069", nil))
}
