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
	"sync"

	"github.com/google/uuid"
)

type FantasmaConfig struct {
	Pub map[string][]string
	Sub map[string]string
	MyAddr string
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

func sub(topic string, payload map[string]interface{}) error {
	payloadBytes, _ := json.Marshal(payload)

	cmd, prs := config.Sub[topic]
	if !prs {
		return nil
	}

	// Write payload to a file to pass to subprocess
	filePath := topic+"-"+uuid.New().String()+".json"
	err := os.WriteFile(filePath, payloadBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file: %s", err.Error())
	}

	// Run subprocess specified in config, passing payload file as first argument
	cmdSegs := strings.Split(cmd, " ")
	cmdSegs = append(cmdSegs, filePath)
	Cmd := exec.Command(cmdSegs[0], cmdSegs[1:]...)
	if err != nil {
		os.Remove(filePath)
		return fmt.Errorf("Failed to start process '" + cmd + "': ", err.Error())
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

	return nil
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
	
	err = sub(topic, body)
	if err != nil {
		w.WriteHeader(500)
	}
}

func pub(topic string, payload map[string]interface{}) error {
	// Get addresses for subscribers to the topic
	addrs, prs := config.Pub[topic]
	if !prs {
		return nil
	}

	// Marshal request body into byte[] for outgoing body
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal body: %s", err.Error())
	}

	// Send to all subscribers
	var wg sync.WaitGroup
	wg.Add(len(addrs))
	for _, addr := range addrs {
		go func(addr string) {
			defer wg.Done()
			res, err := http.Post(addr + "/sub?topic="+topic, "application/json", bytes.NewBuffer(jsonPayload))
			if err != nil {
				fmt.Println("Failed to send to subscriber: ", err.Error())
			} else {
				fmt.Println("Sent to subscriber " + addr + " with response: " + res.Status)
			}
		}(addr)
	}
	wg.Wait()

	// If this node subscribes to the topic, run the subprocess
	err = sub(topic, payload)
	if err != nil {
		return err
	}

	return nil
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
	
	err = pub(topic, body)
	if err != nil {
		w.WriteHeader(500)
	}
}

// Handle request to subscribe to a topic by adding to list of subscribers
func subscribeHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Subscription Recieved Request")

	topic := req.URL.Query().Get("topic")
	addr := req.URL.Query().Get("addr")

	_, prs := config.Pub[topic]
	if prs {
		config.Pub[topic] = append(config.Pub[topic], addr)
		fmt.Fprintf(w, "Subscribed to topic %s", topic)
	} else {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Topic %s not found", topic)
	}
}

// Make a request to all known hosts to subscribe to a topic. Only one should be successful
func subscribeToTopic(topic string, host string) {
	res, err := http.Get(host + "/subscribe?topic="+topic+"&addr="+config.MyAddr)
	if err != nil {
		fmt.Println("Failed to subscribe to topic: ", err.Error())
	} else {
		fmt.Println("Subscribed to topic " + topic + " with response: " + res.Status)
	}
}

func main() {
	config = readConfig(os.Args[1])
	
	port := "2022"
	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	// Subscribe to all topics in config
	var wg sync.WaitGroup
	for topic, publishers := range config.Pub {
		for _, publisher := range publishers {
			go func(topic string, publisher string) {
				defer wg.Done()
				subscribeToTopic(topic, publisher)
			}(topic, publisher)
		}
	}
	wg.Wait()


	http.HandleFunc("/sub", subHandler)
	http.HandleFunc("/pub", pubHandler)
	http.HandleFunc("/subscribe", subscribeHandler)

	fmt.Println("Listening on port "+port+"...")
	http.ListenAndServe(":"+port, nil)
}
