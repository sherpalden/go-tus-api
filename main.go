package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/tus/tusd/pkg/gcsstore"
	tusd "github.com/tus/tusd/pkg/handler"
	"github.com/tus/tusd/pkg/memorylocker"
)

func FileUploadHandler() *tusd.UnroutedHandler {
	// bucket := "your-gcs-bucket-name"
	bucket := "kaki-oki-bucket-dev"
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
	store.ObjectPrefix = "videos" //destination for file object in GCS bucket

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

	router.POST("tus-files/", gin.WrapF(FileUploadHandler().PostFile))
	router.HEAD("tus-files/:id", gin.WrapF(FileUploadHandler().HeadFile))
	router.PATCH("tus-files/:id", gin.WrapF(FileUploadHandler().PatchFile))
	router.GET("tus-files/:id", gin.WrapF(FileUploadHandler().GetFile))

	router.Run(":8080") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
