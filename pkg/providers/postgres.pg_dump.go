package providers

import (
	"fmt"
	"os"

	pg "github.com/habx/pg-commands"
	"github.com/pibblokto/backlokto-worker/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

func PostgresPgDump(jobSpecs map[string]string, targets []map[string]string) {

	var pgPass string
	if jobSpecs["passwordSecretName"] != "" {
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
		secretName := jobSpecs["passwordSecretName"]

		// Get the secret
		secret, err := clientset.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
		if err != nil {
			fmt.Println("Error getting secret:", err)
			os.Exit(1)
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
		Host:     dbHost,
		Port:     dbPort,
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
