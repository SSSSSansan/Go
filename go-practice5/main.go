package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Movie struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Year       int    `json:"year"`
	ActorCount int    `json:"actor_count"`
}

func main() {
	db, err := sql.Open("postgres", "postgres://postgres:password@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/movies", func(w http.ResponseWriter, r *http.Request) {
		handleGetMovies(w, r, db)
	})

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleGetMovies(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	q := r.URL.Query()
	var filters []string
	var args []interface{}
	argID := 1

	if v := q.Get("year_min"); v != "" {
		filters = append(filters, "m.year >= $"+strconv.Itoa(argID))
		args = append(args, v)
		argID++
	}
	if v := q.Get("year_max"); v != "" {
		filters = append(filters, "m.year <= $"+strconv.Itoa(argID))
		args = append(args, v)
		argID++
	}

	limit := 100
	if v := q.Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if v := q.Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil && o >= 0 {
			offset = o
		}
	}

	sqlQuery := "SELECT m.id, m.title, m.year, COUNT(a.id) FROM movies m LEFT JOIN actors a ON a.movie_id = m.id"
	if len(filters) > 0 {
		sqlQuery += " WHERE " + strings.Join(filters, " AND ")
	}
	sqlQuery += " GROUP BY m.id ORDER BY m.year DESC"
	sqlQuery += " LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)

	start := time.Now()
	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	w.Header().Set("X-Query-Time", time.Since(start).String())

	var movies []Movie
	for rows.Next() {
		var m Movie
		if err := rows.Scan(&m.ID, &m.Title, &m.Year, &m.ActorCount); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		movies = append(movies, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movies)
}
