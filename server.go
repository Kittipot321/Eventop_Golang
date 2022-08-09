package main
import (
	"fmt"
	"context"
	"strconv"
	"strings"
	"log"
	"time"
	"net/http"
	"database/sql"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
)

var Db *sql.DB
const eventPath = "events"
const apibasePath = "/api"

type Event struct {
	EventID int
	EventName string
	Description string
	EventDate string
	Location string
	Picture string
	Is_active bool
	Category_id int
}
func SetupDB(){
	var err error
	Db, err = sql.Open("mysql", "root:Kittipot321@tcp(127.0.0.1:3306)/eventdb")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(Db)
	Db.SetConnMaxLifetime(time.Minute * 3)
	Db.SetMaxOpenConns(10)
	Db.SetMaxIdleConns(10)
}

func getEventList() ([]Event, error){
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	results, err := Db.QueryContext(ctx, `SELECT
	* FROM event`)
	if err != nil{
		log.Println(err.Error())
		return nil, err
	}
	defer results.Close()
	events := make([]Event, 0)
	for results.Next(){
		var event Event
		results.Scan(&event.EventID,
			&event.EventName,
			&event.Description,
			&event.EventDate,
			&event.Location,
			&event.Picture,
			&event.Is_active,
			&event.Category_id)
		events = append(events, event)
	}
	return events, nil
}

func insertEvent(event Event) (int, error){
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := Db.ExecContext(ctx, `INSERT INTO event
	(id,
	event_name,
	description,
	event_date,
	location,
	picture,
	is_active,
	category_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		event.EventID,
		event.EventName,
		event.Description,
		event.EventDate,
		event.Location,
		event.Picture,
		event.Is_active,
		event.Category_id)
	if err != nil{
		log.Println(err.Error())
		return 0, err
	}
	insertID, err := result.LastInsertId()
	if err != nil{
		log.Println(err.Error())
		return 0, err
	}
	return int(insertID), nil
}

func getEvent(eventid int) (*Event, error){
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := Db.QueryRowContext(ctx, `SELECT
	* FROM event WHERE id = ?`, eventid)
	event := &Event{}
	err := row.Scan(
		&event.EventID,
		&event.EventName,
		&event.Description,
		&event.EventDate,
		&event.Location,
		&event.Picture,
		&event.Is_active,
		&event.Category_id,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		log.Println(err)
		return nil, err
	}
	return event, nil
}

func editEvent(event Event, eventID int) error{
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := Db.ExecContext(ctx, `UPDATE event SET
	event_name=?,description=?,event_date=?,location=?,picture=?,is_active=?,category_id=? WHERE id=?`,
		event.EventName,
		event.Description,
		event.EventDate,
		event.Location,
		event.Picture,
		event.Is_active,
		event.Category_id,
		eventID)
	if err != nil{
		log.Println(err.Error())
		return err
	}
	return nil
}

func removeEvent(eventID int) error{
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := Db.ExecContext(ctx, `DELETE FROM event WHERE id = ?`, eventID)
	if err != nil{
		log.Println(err.Error())
		return err
	}
	return nil
}

func handleEvents(w http.ResponseWriter, r *http.Request){
	switch r.Method {
	case http.MethodGet:
		eventList, err := getEventList()
		if err != nil{
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		j, err := json.Marshal(eventList)
		if err != nil{
			log.Fatal(err)
		}
		_, err = w.Write(j)
		if err != nil{
			log.Fatal(err)
		}
	case http.MethodPost:
		var event Event
		err := json.NewDecoder(r.Body).Decode(&event)
		if err != nil{
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		eventID, err := insertEvent(event)
		if err != nil{
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf(`ID:%d has created.`, eventID)))
	case http.MethodOptions:
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleEvent(w http.ResponseWriter, r *http.Request) {
	urlPathSegments := strings.Split(r.URL.Path, fmt.Sprintf("%s/", eventPath))
	if len(urlPathSegments[1:]) > 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	eventID, err :=strconv.Atoi(urlPathSegments[len(urlPathSegments)-1])
	if err != nil{
		log.Print(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		event, err := getEvent(eventID)
		if err != nil{
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if event == nil{
			w.WriteHeader(http.StatusNotFound)
			return
		}
		j, err := json.Marshal(event)
		if err != nil{
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err = w.Write(j)
		if err != nil{
			log.Fatal(err)
		}
	case http.MethodPut:
		var event Event
		err := json.NewDecoder(r.Body).Decode(&event)
		if err != nil{
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = editEvent(event, eventID)
		if err != nil{
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Write([]byte(fmt.Sprintf(`ID:%d has updated.`, eventID)))
	case http.MethodDelete:
		err := removeEvent(eventID)
		if err != nil{
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(fmt.Sprintf(`ID:%d has removed.`, eventID)))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func corsMiddleware(handler http.Handler) http.Handler{
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		w.Header().Add("Access-Control-Allow-Origin", "*")
		// w.Header().Add("Access-Control-Allow-Origin", "POST, GET, OPTIONS, PUT, DELETE")
		// w.Header().Add("Access-Control-Allow-Origin", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization,X-CSRF-Token")
		handler.ServeHTTP(w, r)
	})
}

func SetupRoutes(apibasePath string){
	eventsHandler := http.HandlerFunc(handleEvents)
	http.Handle(fmt.Sprintf("%s/%s", apibasePath, eventPath), corsMiddleware(eventsHandler))
	eventHandler := http.HandlerFunc(handleEvent)
	http.Handle(fmt.Sprintf("%s/%s/", apibasePath, eventPath), corsMiddleware(eventHandler))
}

func main() {
	SetupDB()
	SetupRoutes(apibasePath)
	log.Fatal(http.ListenAndServe(":8000", nil))
}