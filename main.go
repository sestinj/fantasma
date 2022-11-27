package main


import (
	"net/http"
	"fmt"
	"os"
	"io/ioutil"
	"encoding/json"
	"os/exec"
	"github.com/google/uuid"
)

func startServer(port string) {
	http.ListenAndServe(":" + port, nil)
}

func handleRequest(conn net.Conn) {
	buf := make([]byte, 1024)
}

func readConfig(path string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Unable to read file at " + path + ": ", err.Error())
		os.Exit(1)
	}
	defer file.Close()

	bytes, _ := ioutil.ReadAll(file)

	var jsonData map[string]interface{}
	json.Unmarshal([]byte(bytes), &jsonData)

	return jsonData
}

const (
	subConfig interface{}
	pubConfig interface{}
)

func subHandler(w http.ResponseWriter, req *http.Request) {
	topic := req.URL.Query().Get("topic")

	cmd, prs := subConfig[topic]
	if !prs {
		fmt.Printf(w, "y")
		return
	}

	// Write payload to a file to pass to subprocess
	filePath := topic+"-"+uuid.New().String()
	err := os.WriteFile(filePath)
	if err != nil {
		fmt.Printf(w, "n")
		return
	}
	
	Cmd := exec.Command(cmd+" "+filePath)
	err:= Cmd.Start()
	if err != nil {
		// Failure to start process, respond 'n'
		// We only check failure to start, not to finish
		fmt.Println("Failed to start process '" + cmd + "': ", err.Error())
		fmt.Printf(w, "n")
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
	fmt.Printf(w, "y")
}

func pubHandler(w http.ResponseWriter, req *http.Request) {
	var body map[string]interface{}
	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		fmt.Println("Failed to decode body: ", err.Error())
		fmt.Printf(w, "n")
		return
	}
	
	ips, prs := pubConfig[req["topic"]]
	if !prs {
		fmt.Printf(w, "y")
		return
	}

	jsonPayload, err := json.Marshal()
	if err != nil {
		fmt.Printf(w, "n")
		return
	}

	// Send to all subscribers
	for var ip in range ips {
		http.Post("http://" + ip + ":3000", "application/json", jsonPayload)
	}

	fmt.Printf(w, "y")
}

func main() {
	config = readConfig(os.Args[1])

	subConfig, sprs := config["sub"]
	pubConfig, pprs := config["pub"]

	http.HandleFunc("/sub", subHandler)
	http.HandleFunc("/pub", pubHandler)

	http.ListenAndServe()
}