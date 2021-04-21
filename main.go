package main

import (
	"crypto/tls"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/mastertinner/adapters/logging"
	"github.com/mastertinner/s3manager/internal/app/s3manager"
	"github.com/matryer/way"
	minio "github.com/minio/minio-go"
)

//go:embed web/template
var templateFS embed.FS

func main() {
	endpoint, ok := os.LookupEnv("ENDPOINT")
	if !ok {
		endpoint = "s3.amazonaws.com"
	}
	accessKeyID, ok := os.LookupEnv("ACCESS_KEY_ID")
	if !ok {
		log.Fatal("please provide ACCESS_KEY_ID")
	}
	secretAccessKey, ok := os.LookupEnv("SECRET_ACCESS_KEY")
	if !ok {
		log.Fatal("please provide SECRET_ACCESS_KEY")
	}
	useSSL := getBoolEnvWithDefault("USE_SSL", true)
	skipSSLVerification := getBoolEnvWithDefault("SKIP_SSL_VERIFICATION", false)
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	// Set up templates
	templates, err := fs.Sub(templateFS, "web/template")
	if err != nil {
		log.Fatal(err)
	}

	// Set up S3 client
	s3, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		log.Fatalln(fmt.Errorf("error creating s3 client: %w", err))
	}
	if useSSL && skipSSLVerification {
		s3.SetCustomTransport(&http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}) //nolint:gosec
	}

	// Set up router
	r := way.NewRouter()
	r.Handle(http.MethodGet, "/", http.RedirectHandler("/buckets", http.StatusPermanentRedirect))
	r.Handle(http.MethodGet, "/buckets", s3manager.HandleBucketsView(s3, templates))
	r.Handle(http.MethodGet, "/buckets/:bucketName", s3manager.HandleBucketView(s3, templates))
	r.Handle(http.MethodPost, "/api/buckets", s3manager.HandleCreateBucket(s3))
	r.Handle(http.MethodDelete, "/api/buckets/:bucketName", s3manager.HandleDeleteBucket(s3))
	r.Handle(http.MethodPost, "/api/buckets/:bucketName/objects", s3manager.HandleCreateObject(s3))
	r.Handle(http.MethodGet, "/api/buckets/:bucketName/objects/:objectName", s3manager.HandleGetObject(s3))
	r.Handle(http.MethodDelete, "/api/buckets/:bucketName/objects/:objectName", s3manager.HandleDeleteObject(s3))

	lr := logging.Handler(os.Stdout)(r)
	log.Fatal(http.ListenAndServe(":"+port, lr))
}

func getBoolEnvWithDefault(name string, defaultValue bool) bool {
	envValue, ok := os.LookupEnv(name)
	if !ok {
		return defaultValue
	}
	value, err := strconv.ParseBool(envValue)
	if err != nil {
		log.Fatalf("invalid value for %s", name)
	}
	return value
}
