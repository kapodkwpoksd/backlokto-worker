package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/pibblokto/backlokto-worker/pkg/providers"
	"github.com/pibblokto/backlokto-worker/pkg/types"
)

var ProvidersMap = map[string]func(map[string]string, []map[string]string){
	"postgres.pg_dump":         providers.PostgresPgDump,
	"mysql.mysqldump":          providers.MysqlDump,
}


func parseTargets(filename string) ([]map[string]string, error) {
	// Read the JSON file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %w", err)
	}

	// Parse JSON data into a slice of maps
	var targets []map[string]string
	err = json.Unmarshal(data, &targets)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON data: %w", err)
	}

	return targets, nil
}

func parseJobSpec(filename string) (map[string]string, error) {
	// Read the JSON file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %w", err)
	}

	// Parse JSON data into a map
	var jobSpec map[string]string
	err = json.Unmarshal(data, &jobSpec)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON data: %w", err)
	}

	return jobSpec, nil
}

// /config/targets.json
// /config/jobSpec.json
func main() {

	// File paths
	targetsFile := "/config/targets.json"
	jobSpecFile := "/config/jobSpec.json"

	// Parse the targets JSON file
	targets, err := parseTargets(targetsFile)
	if err != nil {
		fmt.Printf("Error parsing targets: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Parsed targets: %v\n", targets)

	// Parse the job spec JSON file
	jobSpec, err := parseJobSpec(jobSpecFile)
	if err != nil {
		fmt.Printf("Error parsing job spec: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Parsed job spec: %v\n", jobSpec)				

	// Running BackupJob
	ProvidersMap[jobSpec["provider"]](jobSpec, targets)
	
}