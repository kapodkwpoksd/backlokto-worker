package targets

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/pibblokto/backlokto-worker/pkg/types"
	"google.golang.org/api/option"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
)

func GcsTarget(targetSpecs map[string]string, artifacts *types.Artifacts) {
	// In-cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to local kubeconfig for testing purposes
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatalf("Error creating config: %v", err)
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}
	namespace := os.Getenv("POD_NAMESPACE")
	secretName := targetSpecs["secretName"]
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Error retrieving secret: %v", err)
	}

	// Retrieve the GCP JSON key from the secret
	jsonKey, jsonKeyExists := secret.Data["jsonKey"]

	// Check if the JSON key exists in the secret
	if !jsonKeyExists {
		fmt.Println("GCP JSON key not found in the secret.")
		os.Exit(1)
	}

	// Write the JSON key to a temporary file
	tempFile, err := ioutil.TempFile("", "gcp-credentials-*.json")
	if err != nil {
		log.Fatalf("Error creating temp file for GCP credentials: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(jsonKey); err != nil {
		log.Fatalf("Error writing to temp file for GCP credentials: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		log.Fatalf("Error closing temp file for GCP credentials: %v", err)
	}

	var bucketName string = targetSpecs["bucketName"]
	var filePath string = artifacts.Filepath

	// Create a GCS client
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(tempFile.Name()))
	if err != nil {
		log.Fatalf("Failed to create GCS client: %v", err)
	}
	defer client.Close()

	// Open the file to be uploaded
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Prepare the file for upload
	bucket := client.Bucket(bucketName)
	objectName := createFilename(file.Name())
	if targetSpecs["bucketKey"] != "" {
		objectName = path.Join(targetSpecs["bucketKey"], objectName)
	}
	object := bucket.Object(objectName)
	writer := object.NewWriter(ctx)
	defer writer.Close()

	// Upload the file to GCS
	if _, err := io.Copy(writer, file); err != nil {
		log.Fatalf("Failed to upload file to GCS: %v", err)
	}

	fmt.Printf("File %s uploaded to GCS bucket %s\n", objectName, bucketName)
}
