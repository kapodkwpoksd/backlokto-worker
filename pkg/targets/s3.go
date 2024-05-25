package targets

import (
	"os"

	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pibblokto/backlokto-worker/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"path/filepath"
)

func S3Target(targetSpecs map[string]string, artifacts *types.Artifacts) {
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

	// Retrieve the access key and secret key from the secret
	accessKey, accessKeyExists := secret.Data["accessKey"]
	secretKey, secretKeyExists := secret.Data["secretKey"]

	// Check if both keys exist in the secret
	if !accessKeyExists || !secretKeyExists {
		fmt.Println("Access key or secret key not found in the secret.")
		os.Exit(1)
	}

	// Convert the keys from []byte to string
	var access_key string = string(accessKey)
	var secret_key string = string(secretKey)
	var aws_region string = targetSpecs["awsRegion"]
	var bucket_name string = targetSpecs["bucketName"]
	var filepath string = artifacts.Filepath

	// Create an S3 session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(aws_region),
		Credentials: credentials.NewStaticCredentials(access_key, secret_key, ""),
	})
	if err != nil {
		fmt.Println(fmt.Errorf("failed to create AWS session: %v", err))
		return
	}

	// Open the file to be uploaded
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to open file: %v", err))
		return
	}
	defer file.Close()

	// Create an S3 uploader
	uploader := s3.New(sess)
	var trailing_slash string = ""
	if targetSpecs["bucketKey"] != "" {
		if targetSpecs["bucketKey"][len(targetSpecs["bucketKey"])-1] != '/' {
			trailing_slash = "/"
		}
	}

	// Upload the file to S3
	newFilename := createFilename(file.Name())
	_, err = uploader.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucket_name),
		Key:    aws.String(targetSpecs["bucketKey"] + trailing_slash + newFilename),
		Body:   file,
	})
	if err != nil {
		fmt.Println(fmt.Errorf("failed to upload file to S3: %v", err))
		return
	}

	fmt.Printf("File %s uploaded to S3 bucket %s\n", newFilename, bucket_name)
}
