package config

import (
	"bookings/internal/models"
	"html/template"
	"log"

	"github.com/alexedwards/scs/v2"
)

// AppConfig holds the application config
type AppConfig struct {
	IpAddr        string
	UseCache      bool
	TemplateCache map[string]*template.Template
	InfoLog       *log.Logger
	ErrorLog      *log.Logger
	InProduction  bool
	Session       *scs.SessionManager
	MailChan      chan models.MailData
}
