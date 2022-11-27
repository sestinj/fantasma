package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

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
	fmt.Println("Subscriber Recieved Request")

	topic := req.URL.Query().Get("topic")

	var body map[string]interface{}
	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		fmt.Println("Failed to decode body: ", err.Error())
		fmt.Fprintf(w, "n")
		return
	}
	bodyBytes, _ := json.Marshal(body)

	cmd, prs := subConfig[topic]
	if !prs {
		fmt.Fprintf(w, "y")
		return
	}

	// Write payload to a file to pass to subprocess
	filePath := topic+"-"+uuid.New().String()+".json"
	err = os.WriteFile(filePath, bodyBytes, 0777)
	if err != nil {
		fmt.Println("Failed to write to file: ", err.Error())
		fmt.Fprintf(w, "n")
		return
	}

	cmdSegs := strings.Split(cmd, " ")
	cmdSegs = append(cmdSegs, filePath)
	Cmd := exec.Command(cmdSegs[0], cmdSegs[1:]...)
	// err = Cmd.Start()
	if err != nil {
		// Failure to start process, respond 'n'
		// We only check failure to start, not to finish
		fmt.Println("Failed to start process '" + cmd + "': ", err.Error())
		fmt.Fprintf(w, "n")
		os.Remove(filePath)
		return
	}

	go func() {
		out, err := Cmd.Output()
		if err != nil {
			fmt.Println("Failed to run process '" + cmd + "': ", err.Error())
		}
		fmt.Println("Process '" + cmd + "' finished with output: ", string(out))
		err = os.Remove(filePath)
		if err != nil {
			fmt.Println("Failed to delete file: ", err.Error())
		}
	}()

	// If success, respond with 'y'
	fmt.Fprintf(w, "y")
}

func pubHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Publisher Recieved Request")

	var body map[string]interface{}
	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		fmt.Println("Failed to decode body: ", err.Error())
		fmt.Fprintf(w, "n")
		return
	}
	
	topic := req.URL.Query().Get("topic")

	addrs, prs := pubConfig[topic]
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
	for _, addr := range addrs {
		res, err := http.Post(addr + "/sub?topic="+topic, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			fmt.Println("Failed to send to subscriber: ", err.Error())
			continue
		} else {
			fmt.Println("Sent to subscriber " + addr + " with response: " + res.Status)
		}
	}

	fmt.Fprintf(w, "y")
}

func main() {
	config := readConfig(os.Args[1])
	port := "2022"
	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	subConfig = config.Sub
	pubConfig = config.Pub

	http.HandleFunc("/sub", subHandler)
	http.HandleFunc("/pub", pubHandler)

	fmt.Println("Listening on port "+port+"...")

	http.ListenAndServe(":"+port, nil)
}