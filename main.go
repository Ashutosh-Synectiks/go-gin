package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type PersonInfo struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	City        string `json:"city"`
	State       string `json:"state"`
	Street1     string `json:"street1"`
	Street2     string `json:"street2"`
	ZipCode     string `json:"zip_code"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/cetec")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()
	r.GET("/person/:person_id/info", getPersonInfo)
	r.POST("/person/create", createPerson)
	r.Run(":8080")
}

func getPersonInfo(c *gin.Context) {
	personID := c.Param("person_id")

	var info PersonInfo
	err := db.QueryRow(`
		SELECT p.name, ph.number, a.city, a.state, a.street1, a.street2, a.zip_code
		FROM person p
		JOIN phone ph ON p.id = ph.person_id
		JOIN address_join aj ON p.id = aj.person_id
		JOIN address a ON aj.address_id = a.id
		WHERE p.id = ?`, personID).Scan(&info.Name, &info.PhoneNumber, &info.City, &info.State, &info.Street1, &info.Street2, &info.ZipCode)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, info)
}

func createPerson(c *gin.Context) {
	var req PersonInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result, err := tx.Exec(`INSERT INTO person (name, age) VALUES (?, ?)`, req.Name, 0)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	personID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = tx.Exec(`INSERT INTO phone (number, person_id) VALUES (?, ?)`, req.PhoneNumber, personID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result, err = tx.Exec(`INSERT INTO address (city, state, street1, street2, zip_code) VALUES (?, ?, ?, ?, ?)`, req.City, req.State, req.Street1, req.Street2, req.ZipCode)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	addressID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = tx.Exec(`INSERT INTO address_join (person_id, address_id) VALUES (?, ?)`, personID, addressID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Person created successfully"})
}
