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
	bucket := "your-gcs-bucket-name"
	serviceAccountKeyFilePath, err := filepath.Abs("./serviceAccountKey.json")
	if err != nil {
		log.Fatal("Unable to load serviceAccountKey.json file:: ", err.Error())
	}
	gcs_service, err := gcsstore.NewGCSService(serviceAccountKeyFilePath)
	if err != nil {
		log.Fatal("Unable to create GCS service:: ", err.Error())
	}

	composer := tusd.NewStoreComposer()
	locker := memorylocker.New()

	store := gcsstore.New(bucket, gcs_service)
	store.ObjectPrefix = "tus-files" //destination for file object in GCS bucket

	locker.UseIn(composer)
	store.UseIn(composer)

	handler, err := tusd.NewUnroutedHandler(tusd.Config{
		BasePath:              "/tus-files/",
		StoreComposer:         composer,
		NotifyCompleteUploads: true,
	})
	if err != nil {
		log.Fatal("Unable to create tus handler:: ", err.Error())
	}

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

	router.POST("tus-files/", gin.WrapF(FileUploadHandler().PostFile))
	router.HEAD("tus-files/:id", gin.WrapF(FileUploadHandler().HeadFile))
	router.PATCH("tus-files/:id", gin.WrapF(FileUploadHandler().PatchFile))
	router.GET("tus-files/:id", gin.WrapF(FileUploadHandler().GetFile))

	router.Run(":8080") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
