package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func (a *api) WithMongo(c context.Context, mongoUri string) error {
	ctx, cancel := context.WithTimeout(c, time.Duration(1*time.Second))
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUri))
	if err != nil {
		log.Println("error_connecting_to_mongo : ", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Println("error_pinging_mongo_primary_instance : ", err)
	}

	a.client = client
	return nil
}

type Note struct {
	Id   primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Text string             `json:"text,omitempty" bson:"text,omitempty"`
}

type api struct {
	client *mongo.Client
	ctx    context.Context
}

func (a *api) httpServer() error {
	serverHost := os.Getenv("HOST") // in form 0.0.0.0:8080
	if serverHost == "" {
		serverHost = "0.0.0.0:8080"
	}

	serverPath := os.Getenv("SERVER_PATH") // api/v1/godevops/notes
	if serverPath == "" {
		serverPath = "api/v1/godevops/notes"
	}

	engine := gin.Default()

	engine.GET(serverPath, a.GetNotes)
	engine.POST(serverPath, a.CreateNote)

	server := &http.Server{
		Addr:    serverHost,
		Handler: engine,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalln("error_starting_server : ", err)
	}
	return nil
}

func (a *api) GetNotes(ginctx *gin.Context) {
	notes := []Note{}
	db := a.client.Database("notes_db")
	coll := db.Collection("notes")
	cur, err := coll.Find(a.ctx, bson.M{})
	if err != nil {
		log.Println("error_find_notes : ", err)
		ginctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := cur.All(a.ctx, &notes); err != nil {
		log.Println("error_unmarshling_notes : ", err)
		ginctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ginctx.JSON(http.StatusOK, gin.H{
		"notes": notes,
	})
}

func (a *api) CreateNote(ginctx *gin.Context) {
	req := &Note{}
	if err := ginctx.BindJSON(req); err != nil {
		log.Println("error_binding_request : ", err)
		ginctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	db := a.client.Database("notes_db")
	coll := db.Collection("notes")

	if _, err := coll.InsertOne(a.ctx, req); err != nil {
		log.Println("error_inserting_record : ", err)
		ginctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ginctx.JSON(http.StatusAccepted, gin.H{})
}

func main() {
	api := &api{}
	api.ctx = context.Background()
	mongoUri := os.Getenv("MONGO_URI")
	if mongoUri == "" {
		log.Fatalln("mongo_uri_env_var_is_empty")
	}
	if err := api.WithMongo(api.ctx, mongoUri); err != nil {
		log.Fatalln()
	}

	if err := api.httpServer(); err != nil {
		log.Fatalln()
	}

	log.Println("server_is_up_and_running")
}
