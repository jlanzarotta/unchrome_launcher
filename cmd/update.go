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
	"unchrome_launcher/constants"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/schollz/progressbar/v3"
)

type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

//var p *tea.Program

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

func update(_ *cobra.Command, _ []string) {
	// Step 1: Get latest release info.
	distribution := viper.GetString(constants.CHROME_DISTRIBUTION)
	url := constants.EMPTY
	assetName := constants.EMPTY

	if strings.EqualFold(distribution, constants.UNCHROME_CHROMIUM_DISTRIBUTION) {
		url = constants.UNCHROME_CHROMIUM_WINDOWS_GITHUB_URL
		assetName = constants.UNCHROME_CHROMIUM_WINDOWS_ASSET_NAME
	} else if strings.EqualFold(distribution, constants.UNCHROME_WINCHROME_DISTRIBUTION) {
		url = constants.UNCHROME_WINCHROME_GITHUB_URL
		assetName = constants.UNCHROME_WINCHROME_ASSET_NAME
	} else {
		url = constants.CROMITE_GITHUB_URL
		assetName = constants.CROMITE_ASSET_NAME
	}

	if viper.GetBool(constants.DEBUG) {
		log.Printf("Attempting to Update Distribution[%s] from URL[%s] with assetName[%s]", distribution, url, assetName)
	}

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

	log.Printf("AUTOUPDATING %s to latest release version...\n", distribution)
	log.Println("      Installed Version:", installedVersion)
	log.Println("Latest Released Version:", release.TagName)

	// Step 2: Find the desired asset.
	var downloadURL string
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, assetName) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == constants.EMPTY {
		panic("Asset not found in the latest release!")
	}

	// Find the directory where the Unchrome Launcher executable is located.
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("%s: %v\n",
			color.RedString(constants.FATAL_NORMAL_CASE), err)
		os.Exit(1)
	}
	exeDir := filepath.Dir(exePath)

	if viper.GetBool(constants.DEBUG) {
		log.Printf("ExeDir[%s].", exeDir)
	}

	// Construct the full download path.
	var downloadPath string = filepath.Join(exeDir, viper.GetString(constants.DOWNLOAD_DIRECTORY))

	if viper.GetBool(constants.DEBUG) {
		log.Printf("DownloadDir[%s].", downloadPath)
	}

	filename := filepath.Base(*&downloadURL)
	file, err := os.Create(filepath.Join(downloadPath, filename))
	if err != nil {
		log.Fatalf("%s: could not create file: %s.\n",
			color.RedString(constants.FATAL_NORMAL_CASE), err.Error())
		os.Exit(1)
	}
	defer file.Close() // nolint:errcheck

	// Step 3: Download the asset
	req, _ := http.NewRequest("GET", downloadURL, nil)
	resp, _ = http.DefaultClient.Do(req)
	defer check(resp.Body.Close)

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading",
	)
	io.Copy(io.MultiWriter(file, bar), resp.Body)

	// Construct the full bin path.
	var binPath string = filepath.Join(exeDir, viper.GetString(constants.BIN_DIRECTORY), string(os.PathSeparator))

	log.Printf("Unzipping [%s] into [%s]...", file.Name(), binPath)

	// Step 4: Unzip the contents of the downloaded file to the BIN_DIRECTORY.
	err = unzip(file.Name(), binPath)
	if err != nil {
		log.Fatalf("%s: %s\n",
			color.RedString(constants.FATAL_NORMAL_CASE), err.Error())
		os.Exit(1)
	}

	// Step 5: Write the new version to the configuration file.
	viper.Set(constants.INSTALLED_VERSION, release.TagName)
	viper.WriteConfig()

	log.Printf("Done.\n")

	if viper.GetBool(constants.PAUSE_ON_UPDATE) {
		waitForKeyPress()
	}
}

// check checks the returned error of a function.
func check(f func() error) {
	if err := f(); err != nil {
		fmt.Fprintf(os.Stderr, "received error: %v\n", err)
	}
}
