package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/widhaprasa/go-acme-service/env"
	"github.com/widhaprasa/go-acme-service/middleware"

	certsrepository "github.com/widhaprasa/go-acme-service/repository/certs"
	clientrepository "github.com/widhaprasa/go-acme-service/repository/client"

	certsservice "github.com/widhaprasa/go-acme-service/service/certs"
	clientservice "github.com/widhaprasa/go-acme-service/service/client"

	certscontroller "github.com/widhaprasa/go-acme-service/controller/certs"

	"github.com/gin-gonic/gin"
)

func main() {

	db, err := sql.Open("sqlite3", "db/acme.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	certsRepository := certsrepository.CertsRepository{
		Db: db,
	}
	clientrepository := clientrepository.ClientRepository{
		Db: db,
	}

	clientService := clientservice.ClientService{
		Clientrepository: clientrepository,
	}
	certsService := certsservice.NewCertsService(certsRepository, clientService)

	certsController := &certscontroller.CertsController{
		CertsRepository: certsRepository,
		CertsService:    certsService,
	}

	// Create table
	_, err = certsRepository.CreateTable()
	if err != nil {
		log.Fatal(err)
	}
	_, err = clientrepository.CreateTable()
	if err != nil {
		log.Fatal(err)
	}

	// Initial server time
	ts := time.Now().UnixMilli()

	// Initiate schedule for renew certificates
	certsService.InitRenewSchedule(ts)

	// Initiate schedule for job
	certsService.InitJobSchedule()

	r := gin.New()
	r.Use(gin.Logger())

	r.Use(gin.Recovery())

	// Register handler
	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, map[string]any{
			"status": "ok",
		})
	})
	r.Use(middleware.AuthorizeHeader())
	{
		r.GET("/certs/list", certsController.List)
		r.POST("/certs/privatekey", certsController.GetPrivateKey)
		r.POST("/certs/certificate", certsController.GetCertificate)
		r.POST("/certs/generate", certsController.Generate)
	}

	port := env.SERVICE_PORT
	r.Run(":" + strconv.Itoa(port))
}
