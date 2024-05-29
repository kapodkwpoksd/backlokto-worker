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
	"strings"
	"time"
)

func dumpPostgresDB(pgUsername, pgPassword, pgHost, pgPort string, pgDatabase string) (string, error) {
	pgPassword = strings.TrimSpace(pgPassword)

	// Get the current timestamp
	now := time.Now()
	timestamp := now.Format("0402150206") // Format as mmhhddmmyy

	// Construct the filename with the timestamp
	filename := fmt.Sprintf("%s_%s.dump", pgDatabase, timestamp)

	// Construct the connection string
	connStr := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", pgUsername, pgPassword, pgHost, pgPort, pgDatabase)

	// Construct the pg_dump command with the connection string
	cmd := exec.Command("pg_dump",
		"-F", "p", // custom format
		"-f", filename,
		connStr) // output file

	// Debugging output to verify connection string and command
	fmt.Printf("Using connection string: %s\n", connStr)
	fmt.Printf("Command: %v\n", cmd.Args)

	// Execute the command
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to dump database: %v", err)
	}

	fmt.Println("Database dump created successfully:", filename)
	return filename, nil
}

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

	fileName, err := dumpPostgresDB(pgUsername, pgPass, pgHost, pgPort, pgDatabase)
	if err != nil {
		fmt.Println("Error:", err)
	}

	artifacts := types.NewArtifacts()
	artifacts.Filepath = fileName

	if targets != nil {
		for _, target := range targets {
			TargetsMap[target["type"]](target, artifacts) //maybe use goroutine?
		}
	}

}
