package providers

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/go-sql-driver/mysql"
	"github.com/jamf/go-mysqldump"
	"github.com/pibblokto/backlokto/pkg/types"
)

func MysqlDump(jobSpecs map[string]string, targets []map[string]string) {
	// Open connection to database
	var mysqlPass string
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
