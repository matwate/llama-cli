package cmd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var prompt string
var promptStr = "The code is: "

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Call the llm to answer your questions about your code files",
	Long: `
	You can ask the llm to answer your questions about your code files by calling this command.
`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateArgs(args); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(args[0], args[1])

		fileInput, err := readfileContents(args[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		prompt = args[1]

		fmt.Println("Pleas wait, llm is thinking...")
		output := PrintOutProgress(fileInput)
		if err := WriteToLog(output); err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func validateArgs(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("please provide a file or directory")
	}

	if len(args) > 2 {
		return fmt.Errorf("too many arguments")
	}

	if err := checkFileExists(args[0]); err != nil {
		return fmt.Errorf("file does not exist")

	}

	return nil
}

func checkFileExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist")
	}
	return nil
}

func readfileContents(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("could not open file")
	}
	defer file.Close()

	fileInput := make([]byte, 1000)
	_, err = file.Read(fileInput)
	if err != nil {
		return "", fmt.Errorf("could not read file")
	}
	return string(fileInput), nil
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

func llama3Call(inputFileContent, prompt string) (chan string, chan error) {
	outChan := make(chan string)
	errChan := make(chan error)

	go func() {
		fullPrompt := fmt.Sprintf("%s\n%s", prompt, promptStr+inputFileContent)

		cmd := exec.Command("ollama", "run", "llama3")

		stdin, err := cmd.StdinPipe()
		if err != nil {
			errChan <- fmt.Errorf("Error creating stdin pipe: %v", err)
			return
		}

		go func() {
			defer stdin.Close()
			io.WriteString(stdin, fullPrompt)
		}()

		cmdOut, _ := cmd.StdoutPipe()
		scanner := bufio.NewScanner(cmdOut)

		err = cmd.Start()
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

func WriteToLog(output string) error {
	file, err := os.Create("llama3_output.txt")
	if err != nil {
		return fmt.Errorf("could not create file")
	}
	defer file.Close()

	_, err = file.WriteString(output)
	if err != nil {
		return fmt.Errorf("could not write to file")
	}
	return nil
}
