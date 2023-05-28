package main

import (
	"bookings/internal/config"
	"bookings/internal/driver"
	"bookings/internal/handlers"
	"bookings/internal/helpers"
	"bookings/internal/models"
	"bookings/internal/render"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/v2"
)

const portNumber = ":8080"

// const ipAddr = "172.22.209.191"

var app config.AppConfig
var session *scs.SessionManager
var infoLog *log.Logger
var errorLog *log.Logger

func getLocalHostIpAddr() (error, string) {
	// Get the list of network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("Error:", err)
		return err, ""
	}

	// Iterate over the network interfaces and find the one with the name "eth0"
	for _, iface := range ifaces {
		if iface.Name == "eth0" {
			// Get the list of IP addresses for the interface
			addrs, err := iface.Addrs()
			if err != nil {
				fmt.Println("Error:", err)
				return err, ""
			}

			// Iterate over the IP addresses and find the first one that is an IPv4 address
			for _, addr := range addrs {
				ip, _, err := net.ParseCIDR(addr.String())
				if err != nil {
					fmt.Println("Error:", err)
					return err, ""
				}

				if ip.To4() != nil {
					// Found the first IPv4 address for the "eth0" interface
					fmt.Println("IP address:", ip.String())
					return nil, ip.String()
				}
			}
		}
	}

	return err, ""
}

// main is the main function
func main() {
	db, err := run()
	if err != nil {
		log.Fatal(err)
	}
	defer db.SQL.Close()
	defer close(app.MailChan)
	listenForMail()

	fmt.Println("Starting mail listener")

	err, ipAddr := getLocalHostIpAddr()
	if err != nil {
		log.Fatal("Cannot get localhost IP")
	}
	fmt.Println(fmt.Sprintf("Staring application on Ip %s and port %s", ipAddr, portNumber))
	app.IpAddr = ipAddr

	srv := &http.Server{
		Addr:    ipAddr + portNumber,
		Handler: routes(&app),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func run() (*driver.DB, error) {
	// what am I going to put in the session
	gob.Register(models.Reservation{})
	gob.Register(models.User{})
	gob.Register(models.Room{})
	gob.Register(models.Restriction{})

	mailChan := make(chan models.MailData)
	app.MailChan = mailChan
	// change this to true when in production
	app.InProduction = false

	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	app.InfoLog = infoLog

	errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	app.ErrorLog = errorLog

	// set up the session
	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProduction

	app.Session = session
	// connect to database
	log.Println("Connecting to DB")
	db, err := driver.ConnectSQL("host=169.254.123.57 port=5432 dbname=bookings user=postgres password=dev-station")
	if err != nil {
		log.Fatal("Cannot connect to dB, Dying")
	}

	log.Println("Connected to DB")

	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Fatal("cannot create template cache")
		return nil, err
	}

	app.TemplateCache = tc
	app.UseCache = false

	repo := handlers.NewRepo(&app, db)
	handlers.NewHandlers(repo)
	render.NewRenderer(&app)
	helpers.NewHelpers(&app)

	return db, nil
}
