/*
Copyright © 2018-2025 Jeff Lanzarotta
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

 1. Redistributions of source code must retain the above copyright notice,
    this list of conditions and the following disclaimer.

 2. Redistributions in binary form must reproduce the above copyright notice,
    this list of conditions and the following disclaimer in the documentation
    and/or other materials provided with the distribution.

 3. Neither the name of the copyright holder nor the names of its contributors
    may be used to endorse or promote products derived from this software
    without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
POSSIBILITY OF SUCH DAMAGE.
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"ungoogled_launcher/constants"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	owner     = "ungoogled-software"         // GitHub repo owner
	repo      = "ungoogled-chromium-windows" // GitHub repo name
	tag       = "137.0.7151.68-1.1"          // Target release tag
	assetName = "_windows_x64.zip"           // Name of the asset you want to download
)

type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

var p *tea.Program

type progressWriter struct {
	total      int
	downloaded int
	file       *os.File
	reader     io.Reader
	onProgress func(float64)
}

// versionCmd represents the version command
var updateCmd = &cobra.Command{
	Use: "update",
	Run: func(cmd *cobra.Command, args []string) {
		update(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func (pw *progressWriter) Start() {
	// TeeReader calls pw.Write() each time a new response is received
	_, err := io.Copy(pw.file, io.TeeReader(pw.reader, pw))
	if err != nil {
		p.Send(progressErrMsg{err})
	}
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	pw.downloaded += len(p)
	if pw.total > 0 && pw.onProgress != nil {
		pw.onProgress(float64(pw.downloaded) / float64(pw.total))
	}
	return len(p), nil
}

func getResponse(url string) (*http.Response, error) {
	resp, err := http.Get(url) // nolint:gosec
	if err != nil {
		log.Fatalf("%s: %s\n",
			color.RedString(constants.FATAL_NORMAL_CASE), err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("receiving status of %d for url: %s", resp.StatusCode, url)
	}
	return resp, nil
}

func update(_ *cobra.Command, _ []string) {
	// Step 1: Get latest release info
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		panic(fmt.Sprintf("GitHub API request failed: %s", resp.Status))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		panic(err)
	}

	var installedVersion string = viper.GetString(constants.INSTALLED_VERSION)
	if strings.Compare(installedVersion, release.TagName) == 0 {
		log.Printf("No need to update, you have the latest version[%s] installed.", release.TagName)
		return
	}

	log.Println("AUTOUPDATING to latest release version...")
	log.Println("      Installed Version:", installedVersion)
	log.Println("Latest Released Version:", release.TagName)

	// Step 2: Find the desired asset
	var downloadURL string
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, assetName) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		panic("Asset not found in the latest release")
	}

	// Step 3: Download the asset
	resp, err = getResponse(*&downloadURL)
	if err != nil {
		log.Fatalf("%s: could not get response: %s\n",
			color.RedString(constants.FATAL_NORMAL_CASE), err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close() // nolint:errcheck

	// Don't add TUI if the header doesn't include content size
	// it's impossible see progress without total
	if resp.ContentLength <= 0 {
		log.Fatalf("%s: could not parse content length, aborting download.\n",
			color.RedString(constants.FATAL_NORMAL_CASE))
		os.Exit(1)
	}

	filename := filepath.Base(*&downloadURL)
	file, err := os.Create(filepath.Join(viper.GetString(constants.DOWNLOAD_DIRECTORY), filename))
	if err != nil {
		log.Fatalf("%s: could not create file: %s.\n",
			color.RedString(constants.FATAL_NORMAL_CASE), err.Error())
		os.Exit(1)
	}
	defer file.Close() // nolint:errcheck

	pw := &progressWriter{
		total:  int(resp.ContentLength),
		file:   file,
		reader: resp.Body,
		onProgress: func(ratio float64) {
			p.Send(progressMsg(ratio))
		},
	}

	m := model{
		pw:       pw,
		progress: progress.New(progress.WithSolidFill("#00CC00")),
	}

	// Start Bubble Tea
	p = tea.NewProgram(m)

	// Start the download
	go pw.Start()

	if _, err := p.Run(); err != nil {
		log.Fatalf("%s: error running program: %s.\n",
			color.RedString(constants.FATAL_NORMAL_CASE), err.Error())
		os.Exit(1)
	}

	// Step 4: Unzip the contents of the downloaded file to the BIN_DIRECTORY.
	err = unzip(file.Name(), viper.GetString(constants.BIN_DIRECTORY))
	if err != nil {
		log.Fatalf("%s: %s\n",
			color.RedString(constants.FATAL_NORMAL_CASE), err.Error())
		os.Exit(1)
	}

	// Step 5: Write the new version to the configuration file.
	viper.Set(constants.INSTALLED_VERSION, release.TagName)
	viper.WriteConfig()

	log.Printf("Done.\n");

	if viper.GetBool(constants.PAUSE_ON_UPDATE) {
		waitForKeyPress()
	}
}
