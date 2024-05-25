package providers

import (
	"database/sql"
	"fmt"
	"os"

	"context"
	"github.com/go-sql-driver/mysql"
	"github.com/jamf/go-mysqldump"
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

func MysqlDump(jobSpecs map[string]string, targets []map[string]string) {
	// Open connection to database
	var mysqlPass string
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
		mysqlPass = string(dbPassword)
	} else {
		mysqlPass = jobSpecs["dbPassword"]
	}

	config := mysql.NewConfig()
	config.User = jobSpecs["dbUsername"]
	config.Passwd = mysqlPass
	config.DBName = jobSpecs["dbDatabase"]
	config.Net = "tcp"
	mysqlPort, _ := strconv.Atoi(jobSpecs["dbPort"])
	config.Addr = fmt.Sprintf("%s:%d", jobSpecs["dbHost"], mysqlPort) //"your-hostname:your-port"

	dumpDir := "./"                                        // you should create this directory
	dumpFilenameFormat := fmt.Sprintf("%v", config.DBName) // accepts time layout string and add .sql at the end of file

	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		fmt.Println("Error opening database: ", err)
		return
	}

	// Register database with mysqldump
	dumper, err := mysqldump.Register(db, dumpDir, dumpFilenameFormat)
	if err != nil {
		fmt.Println("Error registering databse:", err)
		return
	}

	fmt.Println()
	fmt.Println(dumpDir)
	fmt.Println(dumpFilenameFormat)
	fmt.Println()
	// Dump database to file
	err = dumper.Dump()
	if err != nil {
		fmt.Println("Error dumping:", err)
		return
	}
	fmt.Println(dumpFilenameFormat)
	fmt.Printf("File is saved to %s.sql\n", dumpFilenameFormat)

	// Close dumper, connected database and file stream.
	dumper.Close()

	artifacts := types.NewArtifacts()
	fmt.Printf("%s%s.sql\n", dumpDir, dumpFilenameFormat)
	artifacts.Filepath = fmt.Sprintf("%s%s.sql", dumpDir, dumpFilenameFormat)

	if targets != nil {
		for _, target := range targets {
			TargetsMap[target["type"]](target, artifacts) //maybe use goroutine?
		}
	}
}
