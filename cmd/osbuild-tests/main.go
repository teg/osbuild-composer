package main

import (
	"encoding/json"
	"log"
	"os/exec"
	"strings"
)

func main() {
	runCommand("compose", "types")
}

func runCommand(command ...string) *json.RawMessage {
	command = append([]string{"--json"}, command...)
	cmd := exec.Command("composer-cli", command...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Colud not create command: " + err.Error())
	}

	log.Printf("$ composer-cli %s\n", strings.Join(command, " "))

	cmd.Start()

	var result *json.RawMessage

	err = json.NewDecoder(stdout).Decode(&result)
	if err != nil {
		log.Fatalf("Could not parse output: " + err.Error())
	}

	err = cmd.Wait()
	if err != nil {
		log.Fatalf("Command failed: " + err.Error())
	}

	bytes, err := result.MarshalJSON()
	if err != nil {
		log.Fatalf("Could not remarshal the recevied JSON: " + err.Error())
	}

	log.Printf("Return:\n%s\n", string(bytes))

	return result
}
