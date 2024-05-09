package targets

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pibblokto/backlokto/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func createFilename(oldName string) string {
	filenameSlice := strings.Split(path.Base(oldName), ".")
	return filenameSlice[0] + "_" + strconv.FormatInt(time.Now().Unix(), 10) + ".sql"
}

func S3Target(targetSpecs map[string]string, articats *types.Artifacts) {
	// Extract values from target and artifacts struct

	// Load in-cluster Kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println("Error loading in-cluster configuration:", err)
		os.Exit(1)
	}

	// Create a Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("Error creating Kubernetes client:", err)
		os.Exit(1)
	}

	// Specify the namespace and secret name
	namespace := os.Getenv("POD_NAMESPACE")
	secretName := targetSpecs["secretName"]

	// Get the secret
	secret, err := clientset.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		fmt.Println("Error getting secret:", err)
		os.Exit(1)
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
	var filepath string = articats.Filepath

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
	if target.S3BucketKey != "" {
		if target.S3BucketKey[len(target.S3BucketKey)-1] != '/' {
			trailing_slash = "/"
		}
	}

	// Upload the file to S3
	newFilename := createFilename(file.Name())
	_, err = uploader.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucket_name),
		Key:    aws.String(target.S3BucketKey + trailing_slash + newFilename),
		Body:   file,
	})
	if err != nil {
		fmt.Println(fmt.Errorf("failed to upload file to S3: %v", err))
		return
	}

	fmt.Printf("File %s uploaded to S3 bucket %s\n", newFilename, bucket_name)
}
