package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type User struct {
	ID      int     `db:"id"`
	Name    string  `db:"name"`
	Email   string  `db:"email"`
	Balance float64 `db:"balance"`
}

func main() {

	dsn := "postgres://user:password@localhost:5430/mydatabase?sslmode=disable"

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		log.Fatalln("Error opening DB:", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalln("Database not reachable:", err)
	}
	fmt.Println(" Connected to database")

	users, err := GetAllUsers(db)
	if err != nil {
		log.Fatalln("Error fetching users:", err)
	}
	fmt.Println("Users:")
	for _, u := range users {
		fmt.Printf("%d. %s — %s (balance: %.2f)\n", u.ID, u.Name, u.Email, u.Balance)
	}

	if err := TransferBalance(db, 1, 2, 20); err != nil {
		log.Println("Transfer failed:", err)
	} else {
		fmt.Println("Transfer completed successfully!")
	}
}

func InsertUser(db *sqlx.DB, u User) error {
	query := `INSERT INTO users (name, email, balance) VALUES (:name, :email, :balance)`
	_, err := db.NamedExec(query, u)
	return err
}

func GetAllUsers(db *sqlx.DB) ([]User, error) {
	var users []User
	err := db.Select(&users, "SELECT id, name, email, balance FROM users")
	return users, err
}

func GetUserByID(db *sqlx.DB, id int) (User, error) {
	var u User
	err := db.Get(&u, "SELECT id, name, email, balance FROM users WHERE id=$1", id)
	return u, err
}

func TransferBalance(db *sqlx.DB, fromID, toID int, amount float64) error {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	var from User
	if err := tx.Get(&from, "SELECT * FROM users WHERE id=$1 FOR UPDATE", fromID); err != nil {
		tx.Rollback()
		return fmt.Errorf("get sender: %w", err)
	}

	var to User
	if err := tx.Get(&to, "SELECT * FROM users WHERE id=$1 FOR UPDATE", toID); err != nil {
		tx.Rollback()
		return fmt.Errorf("get receiver: %w", err)
	}

	if amount <= 0 {
		tx.Rollback()
		return fmt.Errorf("amount must be positive")
	}
	if from.Balance < amount {
		tx.Rollback()
		return fmt.Errorf("insufficient funds")
	}

	if _, err := tx.Exec("UPDATE users SET balance = balance - $1 WHERE id = $2", amount, fromID); err != nil {
		tx.Rollback()
		return fmt.Errorf("update sender: %w", err)
	}
	if _, err := tx.Exec("UPDATE users SET balance = balance + $1 WHERE id = $2", amount, toID); err != nil {
		tx.Rollback()
		return fmt.Errorf("update receiver: %w", err)
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}
