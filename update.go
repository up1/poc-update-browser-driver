package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/manifoldco/promptui"
	"robpike.io/filter"
)

var selectedVersion = ""

func main() {
	selectedVersion = getChromeVersion()
	fmt.Printf("Current Google Chrome Browser = %s", selectedVersion)
	url := getUrlOfChromeDriver(selectedVersion)
	if error := downloadFile(url, "./chromedriver_mac.zip"); error != nil {
		fmt.Printf("Got error %v", error)
	}
	fmt.Println("Download Finished")
}

func getChromeVersion() string {
	cmd := exec.Command("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	version := strings.Fields(out.String())
	majorVersion := strings.Split(version[2], ".")
	return fmt.Sprintf("%s", majorVersion[0])
}

var myClient = &http.Client{Timeout: 10 * time.Second}

type Item struct {
	Items []Driver `json:"items"`
}

type Driver struct {
	Name      string `json:"name"`
	MediaLink string `json:"mediaLink"`
}

func getUrlOfChromeDriver(version string) string {
	items := &Item{}
	error := getJson("https://www.googleapis.com/storage/v1/b/chromedriver/o/", items)
	if error != nil {
		fmt.Errorf("Error %v", error)
	}
	results := filter.Choose(items.Items, isHasVersion)
	url := chooseDriver(results)
	return url
}

func isHasVersion(driver Driver) bool {
	return strings.HasPrefix(driver.Name, selectedVersion) && strings.Contains(driver.Name, "mac")
}

func chooseDriver(drivers interface{}) string {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }} ?",
		Active:   "\U0001F336 {{ .Name | cyan }}",
		Inactive: "  {{ .Name | cyan }}",
		Selected: "\U0001F336 {{ .Name | red | cyan }}",
	}
	prompt := promptui.Select{
		Label:     "Select Chrome Driver",
		Items:     drivers,
		Templates: templates,
	}

	i, _, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return ""
	}
	selected := drivers.([]Driver)
	return selected[i].MediaLink
}

func getJson(url string, target interface{}) error {
	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

// WriteCounter counts the number of bytes written to it. By implementing the Write method,
// it is of the io.Writer interface and we can pass this into io.TeeReader()
// Every write to this writer, will print the progress of the file write.
type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

// PrintProgress prints the progress of a file write
func (wc WriteCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 50))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}

// DownloadFile will download a url and store it in local filepath.
// It writes to the destination file as it downloads it, without
// loading the entire file into memory.
// We pass an io.TeeReader into Copy() to report progress on the download.
func downloadFile(url string, filepath string) error {

	// Create the file with .tmp extension, so that we won't overwrite a
	// file until it's downloaded fully
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create our bytes counter and pass it to be used alongside our writer
	counter := &WriteCounter{}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Println()

	// Rename the tmp file back to the original file
	err = os.Rename(filepath+".tmp", filepath)
	if err != nil {
		return err
	}

	return nil
}
