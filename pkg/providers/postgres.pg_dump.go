package providers

import (
	"fmt"
	"os"

	"context"
	pg "github.com/habx/pg-commands"
	"github.com/pibblokto/backlokto-worker/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"path/filepath"
	"strconv"
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
	pgPort, _ := strconv.Atoi(jobSpecs["dbPort"])
	pgDatabase := jobSpecs["dbDatabase"]

	dump, err := pg.NewDump(&pg.Postgres{
		Host:     pgHost,
		Port:     pgPort,
		DB:       pgDatabase,
		Username: pgUsername,
		Password: pgPass,
	})

	if err != nil {
		fmt.Println(err)
	}

	dump.SetFileName(fmt.Sprintf(`%v.sql`, dump.DB))
	dump.SetupFormat("p")

	dumpExec := dump.Exec(pg.ExecOptions{StreamPrint: true})
	if dumpExec.Error != nil {
		fmt.Println(dumpExec.Error.Err)
		fmt.Println(dumpExec.Output)
	} else {
		fmt.Printf("Dump was succesfull. Filename: %s\n", dumpExec.File)
		fmt.Println(dumpExec.File)
	}

	artifacts := types.NewArtifacts()
	artifacts.Filepath = dumpExec.File

	if targets != nil {
		for _, target := range targets {
			TargetsMap[target["type"]](target, artifacts) //maybe use goroutine?
		}
	}

}
