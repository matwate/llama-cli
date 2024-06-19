package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

const promptStr = "the code is:"

var (
	file   string
	prompt string
)

func parseCmdLineArgs() (string, string) {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <file> [prompt]", os.Args[0])
		return "", ""
	}
	switch len(os.Args) {
	case 2:
		file = os.Args[1]
		prompt = ""
	case 3:
		file = os.Args[1]
		prompt = os.Args[2]
	default:
		log.Fatalf("Usage: %s <file> [prompt]", os.Args[0])
		return "", ""
	}
	return file, prompt
}

func checkFileExists(file string) error {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("File %s does not exist", file)
	}
	return nil
}

func readfileContent(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("Error reading file %s: %v", file, err)
	}
	return string(data), nil
}

func llama3Call(inputFileContent, prompt string) (chan string, chan error) {
	outChan := make(chan string)
	errChan := make(chan error)

	go func() {
		fullPrompt := fmt.Sprintf("%s\n%s", prompt, promptStr+inputFileContent)

		cmd := exec.Command("ollama", "run", "llama3", fullPrompt)
		cmdOut, _ := cmd.StdoutPipe()
		scanner := bufio.NewScanner(cmdOut)

		err := cmd.Start()
		if err != nil {
			errChan <- fmt.Errorf("Error starting llama3: %v", err)
			return
		}

		go func() {
			for scanner.Scan() {
				outChan <- scanner.Text()
			}
			if err := scanner.Err(); err != nil {
				errChan <- fmt.Errorf("Error reading output: %v", err)
			}
			close(outChan)
		}()

		err = cmd.Wait()
		if err != nil {
			errChan <- fmt.Errorf("Error running llama3: %v", err)
		}
		close(errChan)
	}()

	return outChan, errChan
}

func WriteToLog(content string) error {
	err := ioutil.WriteFile("output.log", []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("Error writing to file: %v", err)

	}
	return nil
}

func PrintOutProgress(data string) string {

	var result []string

	outChan, errChan := llama3Call(data, prompt)
	for outChan != nil || errChan != nil {
		select {
		case line, ok := <-outChan:
			if !ok {
				outChan = nil
				continue
			}
			fmt.Println(line)
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
				continue
			}
			if err != nil {
				log.Fatalf(err.Error())
				os.Exit(1)
			}
		}
	}
	output := strings.Join(result, "\n")

	return output
}

func main() {
	file, prompt = parseCmdLineArgs()

	if err := checkFileExists(file); err != nil {
		log.Fatalf(err.Error())
		os.Exit(1)
	}

	data, err := readfileContent(file)
	if err != nil {
		log.Fatalf(err.Error())
		os.Exit(1)
	}

	fmt.Println("Please wait, processing...")
	output := PrintOutProgress(data)

	if err := WriteToLog(output); err != nil {
		log.Fatalf(err.Error())
		os.Exit(1)
	}

}
