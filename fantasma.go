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

type FantasmaConfig struct {
	Pub map[string][]string
	Sub map[string]string
}

var config FantasmaConfig

// Read configuration file into FantasmaConfig struct
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
		w.WriteHeader(500)
		return
	}
	bodyBytes, _ := json.Marshal(body)

	cmd, prs := config.Sub[topic]
	if !prs {
		return
	}

	// Write payload to a file to pass to subprocess
	filePath := topic+"-"+uuid.New().String()+".json"
	err = os.WriteFile(filePath, bodyBytes, 0777)
	if err != nil {
		fmt.Println("Failed to write to file: ", err.Error())
		w.WriteHeader(500)
		return
	}

	// Run subprocess specified in config, passing payload file as first argument
	cmdSegs := strings.Split(cmd, " ")
	cmdSegs = append(cmdSegs, filePath)
	Cmd := exec.Command(cmdSegs[0], cmdSegs[1:]...)
	if err != nil {
		fmt.Println("Failed to start process '" + cmd + "': ", err.Error())
		w.WriteHeader(500)
		os.Remove(filePath)
		return
	}

	// When process finishes, remove payload file and print output
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
}

func pubHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Publisher Recieved Request")

	// Get query string and decode request body
	topic := req.URL.Query().Get("topic")
	
	var body map[string]interface{}
	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		fmt.Println("Failed to decode body: ", err.Error())
		w.WriteHeader(500)
		return
	}
	
	// Get addresses for subscribers to the topic
	addrs, prs := config.Pub[topic]
	if !prs {
		return
	}

	// Marshal request body into byte[] for outgoing body
	jsonPayload, err := json.Marshal(body)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	// Send to all subscribers
	go func() {
		for _, addr := range addrs {
			res, err := http.Post(addr + "/sub?topic="+topic, "application/json", bytes.NewBuffer(jsonPayload))
			if err != nil {
				fmt.Println("Failed to send to subscriber: ", err.Error())
				continue
			} else {
				fmt.Println("Sent to subscriber " + addr + " with response: " + res.Status)
			}
		}
	}()
}

func main() {
	config = readConfig(os.Args[1])

	port := "2022"
	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	http.HandleFunc("/sub", subHandler)
	http.HandleFunc("/pub", pubHandler)

	fmt.Println("Listening on port "+port+"...")
	http.ListenAndServe(":"+port, nil)
}