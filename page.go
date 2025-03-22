package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/joho/godotenv"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type Page struct {
	Title string
	Body  []byte
}

type Server struct {
	db *sql.DB
}

func (s *Server) loadPage(title string) (*Page, error) {
	row := s.db.QueryRow(`SELECT title, body FROM pages WHERE title = ?`, title)
	var p Page
	if err := row.Scan(&p.Title, &p.Body); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tato stranka neexistuje")
		}
		return nil, fmt.Errorf("nenasiel som stranku: %w", err)
	}
	return &p, nil
}

func (s *Server) createOrUpdatePage(p *Page) error {
	_, err := s.db.Exec(`
		INSERT INTO pages (title, body)
		VALUES (?, ?)
		ON CONFLICT(title)
		DO UPDATE SET body = excluded.body
	`, p.Title, p.Body)
	return err
}

func (s *Server) deletePage(title string) error {
	log.Println("Deleting page", title)
	_, err := s.db.Exec(`DELETE FROM pages WHERE title = ?`, title)
	return err
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	filename := tmpl + ".html"
	t, err := template.ParseFiles(filepath.Join("templates", filename))
	if err != nil {
		http.Error(w, fmt.Sprintf("template neexistuje: %v", err), http.StatusInternalServerError)
		return
	}
	if err := t.Execute(w, p); err != nil {
		http.Error(w, fmt.Sprintf("syntaxicka chyba v: %v", err), http.StatusInternalServerError)
	}
}

func parseTitle(r *http.Request, prefix string) string {
	return r.URL.Path[len(prefix):]
}

func (s *Server) viewHandler(w http.ResponseWriter, r *http.Request) {
	title := parseTitle(r, "/view/")
	p, err := s.loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func (s *Server) editHandler(w http.ResponseWriter, r *http.Request) {
	title := parseTitle(r, "/edit/")
	p, err := s.loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func (s *Server) saveHandler(w http.ResponseWriter, r *http.Request) {
	oldTitle := parseTitle(r, "/save/")
	newTitle := r.FormValue("name")
	body := r.FormValue("body")
	if oldTitle == "new" {
		oldTitle = newTitle
	}

	p := &Page{Title: newTitle, Body: []byte(body)}
	if err := s.createOrUpdatePage(p); err != nil {
		http.Error(w, fmt.Sprintf("neulozil som stranku: %v", err), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+newTitle, http.StatusFound)
}

func (s *Server) listHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`SELECT title FROM pages`)
	if err != nil {
		http.Error(w, fmt.Sprintf("nenasiel som stranky: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var titles []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			http.Error(w, fmt.Sprintf("nazov neexistuje: %v", err), http.StatusInternalServerError)
			return
		}
		titles = append(titles, t)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<form action="/edit/new" method="get">
		           <input type="submit" value="Create note"/>
		           </form>`)
	fmt.Fprint(w, `<ul>`)
	for _, title := range titles {
		fmt.Fprintf(w, `<li>%s - <a href="/view/%s">View</a> | <a href="/delete/%s">Delete</a></li>`,
			title, title, title)
	}
	fmt.Fprint(w, `</ul>`)
}

func (s *Server) deleteHandler(w http.ResponseWriter, r *http.Request) {
	title := parseTitle(r, "/delete/")
	log.Println("Deleting page", title)
	if err := s.deletePage(title); err != nil {
		http.Error(w, fmt.Sprintf("neviem vymazat entry: %v", err), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func main() {
	if err := godotenv.Load(); err != nil {
		return
	}
	dbURL := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	connString := fmt.Sprintf("%s?authToken=%s", dbURL, authToken)
	db, err := sql.Open("libsql", connString)
	if err != nil {
		return
	}
	defer db.Close()
	s := &Server{db: db}
	http.HandleFunc("/view/", s.viewHandler)
	http.HandleFunc("/edit/", s.editHandler)
	http.HandleFunc("/save/", s.saveHandler)
	http.HandleFunc("/delete/", s.deleteHandler)
	http.HandleFunc("/", s.listHandler)
	log.Fatal(http.ListenAndServe(":42069", nil))
}
