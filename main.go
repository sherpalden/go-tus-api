package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/tus/tusd/pkg/gcsstore"
	tusd "github.com/tus/tusd/pkg/handler"
	"github.com/tus/tusd/pkg/memorylocker"
)

func FileUploadHandler() *tusd.UnroutedHandler {
	//GCS bucket is for storing uploaded files
	bucket := "your-gcs-bucket-name"

	/*
		It is a json file, that contains the credentials which allows to perform actions
		for a specific resource in our GCP project. While creating service account, we should assign
		a set of roles and permissions to the service account that allows to do specific actions.
		In our case, this service account should have the permission to read, write and modify objects in GCS bucket.
	*/
	serviceAccountKeyFilePath, err := filepath.Abs("./serviceAccountKey.json")
	if err != nil {
		log.Fatal("Unable to load serviceAccountKey.json file:: ", err.Error())
	}

	//initialize GCS service
	gcs_service, err := gcsstore.NewGCSService(serviceAccountKeyFilePath)
	if err != nil {
		log.Fatal("Unable to create GCS service:: ", err.Error())
	}

	//StoreComposer represents a composable data store. It consists of the core data store and optional extensions.
	composer := tusd.NewStoreComposer()

	/*
		Memorylocker provides an in-memory locking mechanism.
		When multiple processes are attempting to access an upload, whether it be by reading or writing,
		a synchronization mechanism is required to prevent data corruption, especially to ensure correct offset
		values and the proper order of chunks inside a single upload.
		MemoryLocker persists locks using memory and therefore allowing a simple and cheap mechanism.
		Locks will only exist as long as this object is kept in reference and will be erased if the program exits.
	*/
	locker := memorylocker.New()
	locker.UseIn(composer) //Add memory locker extension to the store composer

	/*
		GCSStore is a storage backend that uses the GCSAPI interface in order to store uploads on GCS.
		Uploads will be represented by two files in GCS; the data file will be stored as an extensionless
		object uid and the JSON info file will stored as uid.info.
	*/
	store := gcsstore.New(bucket, gcs_service)
	store.ObjectPrefix = "tus-files" //specify the destination folder in GCS bucket where the files are to be uploaded
	store.UseIn(composer)            // Add gcs store extension to the composer

	//create a handler for handling fileupload operations.
	handler, err := tusd.NewUnroutedHandler(tusd.Config{
		BasePath:              "/tus-files/", //this defines a base path for fileupload endpoint.
		StoreComposer:         composer,
		NotifyCompleteUploads: true,
	})
	if err != nil {
		log.Fatal("Unable to create tus handler:: ", err.Error())
	}

	//go routine to check for upload complete and notify once it is completed.
	go func() {
		for {
			event := <-handler.CompleteUploads
			fmt.Printf("\nSuccessfully uploaded a file with an id: %v", event.Upload.ID)
		}
	}()

	return handler
}

func main() {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3000"},
		AllowMethods: []string{"PUT", "PATCH", "GET", "POST", "OPTIONS", "DELETE"},
		//these headers are tus specific headers that should be allowed and exposed for CORS while uploading file.
		AllowHeaders: []string{
			"Access-Control-Allow-Origin",
			"Access-Control-Allow-Headers",
			"X-HTTP-Method-Override",
			"Upload-Length",
			"Upload-Offset",
			"Tus-Resumable",
			"Upload-Metadata",
			"Upload-Defer-Length",
			"Upload-Concat",
			"User-Agent",
			"Referrer",
			"Origin",
			"Content-Type",
			"Content-Length",
			"Content-Range",
			"Accept-Encoding",
			"Accept",
			"Cache-Control",
			"X-Agent-User",
			"X-Requested-With",
		},
		ExposeHeaders: []string{
			"Upload-Offset",
			"Location",
			"Tus-Version",
			"Tus-Resumable",
			"Tus-Max-Size",
			"Tus-Extension",
			"Upload-Metadata",
			"Upload-Defer-Length",
			"Upload-Concat",
			"Upload-Offset",
			"Upload-Length",
		},
	}))

	//these endpoints handle the fileupload requests.
	router.POST("tus-files/", gin.WrapF(FileUploadHandler().PostFile))
	router.HEAD("tus-files/:id", gin.WrapF(FileUploadHandler().HeadFile))
	router.PATCH("tus-files/:id", gin.WrapF(FileUploadHandler().PatchFile))
	router.GET("tus-files/:id", gin.WrapF(FileUploadHandler().GetFile))

	router.Run(":8080") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
