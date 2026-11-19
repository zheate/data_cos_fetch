package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

func main() {
	var request struct {
		Mode    string   `json:"mode"`
		Entries []string `json:"entries"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	handle, _, _ := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleWindow").Call()
	if handle != 0 {
		fmt.Fprintln(os.Stderr, "extraction worker has a console window")
		os.Exit(3)
	}
	switch request.Mode {
	case "failure":
		fmt.Fprintln(os.Stderr, "worker diagnostic")
		os.Exit(23)
	case "wait":
		time.Sleep(10 * time.Second)
	}
	result := map[string]any{"total": 1, "records": []map[string]string{{"entry_id": request.Entries[0]}}}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		os.Exit(4)
	}
}
