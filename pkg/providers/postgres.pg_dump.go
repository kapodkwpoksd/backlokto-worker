package providers

import (
	"context"
	"fmt"
	"github.com/pibblokto/backlokto-worker/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func PostgresPgDump(jobSpecs map[string]string, targets []map[string]string) {

	var pgPass string
	if jobSpecs["passwordSecretName"] != "" {
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

		// Specify the namespace and secret name

		secretName := jobSpecs["passwordSecretName"]
		namespace := os.Getenv("POD_NAMESPACE")

		secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil {
			log.Fatalf("Error retrieving secret: %v", err)
		}

		// Retrieve the password
		dbPassword, dbPasswordKeyExists := secret.Data["dbPassword"]
		fmt.Printf("db password in secret %s is: %s\n", secretName, dbPassword)

		// Check if both keys exist in the secret
		if !dbPasswordKeyExists {
			fmt.Println("Database password not found in the secret.")
			os.Exit(1)
		}
		pgPass = string(dbPassword)
	} else {
		pgPass = jobSpecs["dbPassword"]
	}
	pgUsername := jobSpecs["dbUsername"]
	pgHost := jobSpecs["dbHost"]
	pgPort := jobSpecs["dbPort"]
	pgDatabase := jobSpecs["dbDatabase"]

	// Get the current timestamp
	now := time.Now()
	timestamp := now.Format("0402150206")
	filename := fmt.Sprintf("%s_%s.dump", pgDatabase, timestamp)

	cmd := exec.Command("pg_dump",
		"-U", pgUsername,
		"-h", pgHost,
		"-p", pgPort,
		"-d", pgDatabase,
		"-F", "p", // custom format
		"-f", filename) // output file

	// Set the environment variables for the command
	os.Setenv("PGPASSWORD", pgPass)
	//cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", pgPass))

	// Set the environment variable for the password if necessary

	// Execute the command
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("failed to dump database: %v", err)
	} else {
		fmt.Println("Database dump created successfully:", filename)
	}

	artifacts := types.NewArtifacts()
	artifacts.Filepath = fmt.Sprintf(filename, pgDatabase, timestamp)

	if targets != nil {
		for _, target := range targets {
			TargetsMap[target["type"]](target, artifacts) //maybe use goroutine?
		}
	}

}
