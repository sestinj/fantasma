package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/google/uuid"
)

type SubConfig map[string]string
type PubConfig map[string][]string
type FantasmaConfig struct {
	Pub PubConfig
	Sub SubConfig
}

var (
	subConfig SubConfig
	pubConfig PubConfig
)

func readConfig(path string) FantasmaConfig {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Unable to read file at " + path + ": ", err.Error())
		os.Exit(1)
	}
	defer file.Close()

	bytes, _ := ioutil.ReadAll(file)

	var jsonData FantasmaConfig
	json.Unmarshal([]byte(bytes), &jsonData)

	return jsonData
}

func subHandler(w http.ResponseWriter, req *http.Request) {
	topic := req.URL.Query().Get("topic")

	var body []byte
	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		fmt.Println("Failed to decode body: ", err.Error())
		fmt.Fprintf(w, "n")
		return
	}

	cmd, prs := subConfig[topic]
	if !prs {
		fmt.Fprintf(w, "y")
		return
	}

	// Write payload to a file to pass to subprocess
	filePath := topic+"-"+uuid.New().String()
	err = os.WriteFile(filePath, body, 0644)
	if err != nil {
		fmt.Fprintf(w, "n")
		return
	}
	
	Cmd := exec.Command(cmd+" "+filePath)
	err = Cmd.Start()
	if err != nil {
		// Failure to start process, respond 'n'
		// We only check failure to start, not to finish
		fmt.Println("Failed to start process '" + cmd + "': ", err.Error())
		fmt.Fprintf(w, "n")
		return
	}

	go func() {
		err := Cmd.Wait()
		err = os.Remove(filePath)
		if err != nil {
			fmt.Println("Failed to delete file: ", err.Error())
		}
	}()

	// If success, respond with 'y'
	fmt.Fprintf(w, "y")
}

func pubHandler(w http.ResponseWriter, req *http.Request) {
	var body map[string]interface{}
	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		fmt.Println("Failed to decode body: ", err.Error())
		fmt.Fprintf(w, "n")
		return
	}
	
	topic := req.URL.Query().Get("topic")

	ips, prs := pubConfig[topic]
	if !prs {
		fmt.Fprintf(w, "y")
		return
	}

	jsonPayload, err := json.Marshal(body)
	if err != nil {
		fmt.Fprintf(w, "n")
		return
	}

	// Send to all subscribers
	for _, ip := range ips {
		http.Post("http://" + ip + ":3000", "application/json", bytes.NewBuffer(jsonPayload))
	}

	fmt.Fprintf(w, "y")
}

func main() {
	config := readConfig(os.Args[1])

	subConfig = config.Sub
	pubConfig = config.Pub

	http.HandleFunc("/sub", subHandler)
	http.HandleFunc("/pub", pubHandler)

	http.ListenAndServe(":2022", nil)
}