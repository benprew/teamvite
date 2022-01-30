package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

var fMap = template.FuncMap{
	"gravatarKey":  gravatarKey,
	"urlFor":       urlFor,
	"playerEmails": playerEmails,
	"ReminderUrl":  ReminderUrl,
	"ReminderId":   ReminderId,
	"CalendarUrl":  CalendarUrl,
	"Telify":       Telify,
}

type LayoutData struct {
	Message string
	User    player
	Page    interface{} // page specific parameters
}

//go:embed views
var views embed.FS

// content is the template string
func LoadContentTemplate(filename string) (*template.Template, error) {
	return template.Must(templates.Clone()).ParseFS(views, filename)
}

func (s *server) RenderTemplate(w http.ResponseWriter, r *http.Request, template string, templateParams interface{}) error {
	tmpl, err := LoadContentTemplate(template)
	if err != nil {
		return err
	}

	u := s.GetUser(r)
	msg := s.GetMessage(u)
	log.Printf("Showing message: %s\n", msg)

	params := LayoutData{
		Message: msg,
		User:    u,
		Page:    templateParams,
	}

	log.Println("Executing layout.tmpl")
	if err := tmpl.ExecuteTemplate(w, "layout.tmpl", params); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

var templates = defaultTemplates()

func defaultTemplates() *template.Template {
	tmpl := template.Must(
		template.New("layout.tmpl").Funcs(fMap).ParseFS(views, "views/layout.tmpl"))

	return template.Must(tmpl.ParseFS(views, "views/partials/*.tmpl"))
}

func CalendarUrl(t Item) template.URL {
	return template.URL(fmt.Sprintf(
		"webcal://%s/team/%d/calendar.ics",
		CONFIG.Servername, t.itemId()))
}
