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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"ungoogled_launcher/constants"

	"github.com/eiannone/keyboard"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowText            = user32.NewProc("GetWindowTextW")
	procGetWindowTextLength      = user32.NewProc("GetWindowTextLengthW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procShowWindow               = user32.NewProc("ShowWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procSendMessage              = user32.NewProc("SendMessageW")
)

const SW_HIDE = 0            // Hides the window and activates another window.
const SW_SHOWNORMAL = 1      // Activates and displays a window. If the window is minimized, maximized, or arranged, the system restores it to its original size and position. An application should specify this flag when displaying the window for the first time.
const SW_SHOWMINIMIZED = 2   // Activates the window and displays it as a minimized window.
const SW_SHOWMAXIMIZED = 3   // Activates the window and displays it as a maximized window.
const SW_SHOWNOACTIVE = 4    // Displays a window in its most recent size and position. This value is similar to SW_SHOWNORMAL, except that the window is not activated.
const SW_SHOW = 5            // Activates the window and displays it in its current size and position.
const SW_MINIMIZE = 6        // Minimizes the specified window and activates the next top-level window in the Z order.
const SW_SHOWMINNOACTIVE = 7 // Displays the window as a minimized window. This value is similar to SW_SHOWMINIMIZED, except the window is not activated.
const SW_SHOWNA = 8          // Displays the window in its current size and position. This value is similar to SW_SHOW, except that the window is not activated.
const SW_RESTORE = 9         // Activates and displays the window. If the window is minimized, maximized, or arranged, the system restores it to its original size and position. An application should specify this flag when restoring a minimized window.
const SW_SHOWDEFAULT = 10    // Sets the show state based on the SW_ value specified in the STARTUPINFO structure passed to the CreateProcess function by the program that started the application.
const SW_FORCEMINIMIZE = 11  // Minimizes a window, even if the thread that owns the window is not responding. This flag should only be used when minimizing windows from a different thread.

var runCmd = &cobra.Command{
	Use: "run",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func run(_ *cobra.Command, args []string) {
	// Then we run.
	// Command and its arguments

	// Find the directory where the Ungoogled Launcher executable is located.
    exePath, err := os.Executable()
    if err != nil {
		log.Fatalf("%s: %v\n",
			color.RedString(constants.FATAL_NORMAL_CASE), err)
		os.Exit(1)
    }
    exeDir := filepath.Dir(exePath)

	if viper.GetBool(constants.DEBUG) {
    	log.Println("Executable directory:", exeDir)
	}

	var path string = filepath.Join(exeDir, viper.GetString(constants.BIN_DIRECTORY), "chrome.exe")
	path = filepath.Clean(path)
	var profileDirectory string = filepath.Join(exeDir, viper.GetString(constants.PROFILE_DIRECTORY))
	var finalArguments []string = strings.Split(viper.GetString(constants.CHROME_COMMAND_LINE_OPTIONS), constants.SPACE)
	finalArguments = append(finalArguments, "--user-data-dir="+profileDirectory)
	var newArgs = processArgs(args)
	finalArguments = append(finalArguments, newArgs...)

	runChrome(path, finalArguments)
	findAndFocusWindowBySubstring("Chromium")

	if viper.GetBool(constants.PAUSE_AFTER_RUN) {
		waitForKeyPress()
	}
}

func findAndFocusWindowBySubstring(substring string) bool {
	found := false

	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		length, _, _ := procGetWindowTextLength.Call(hwnd)
		if length == 0 {
			return 1 // continue
		}

		buf := make([]uint16, length+1)
		procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), length+1)
		title := syscall.UTF16ToString(buf)

		if strings.Contains(strings.ToLower(title), strings.ToLower(substring)) {
			if viper.GetBool(constants.DEBUG) {
				log.Printf("Found window: \"%s\" (HWND: 0x%X)\n", title, hwnd)
			}

			// Bring to foreground.
			procShowWindow.Call(hwnd, SW_SHOWNA)
			procSetForegroundWindow.Call(hwnd)

			found = true
			return 0 // stop enumeration
		}

		return 1 // continue
	})

	procEnumWindows.Call(cb, 0)
	return found
}

func runChrome(path string, arguments []string) {
	cmd := exec.Command(path, arguments...)
	err := cmd.Start()
	if err != nil {
		log.Fatalf("%s: [%v]\n",
			color.RedString(constants.FATAL_NORMAL_CASE), err)
	} else {
		if viper.GetBool(constants.DEBUG) {
			log.Printf("Started process (PID: %d)\n", cmd.Process.Pid)
		}
	}
}

func processArgs(args []string) []string {
	var newArgs []string

	for _, arg := range args {
		if strings.HasPrefix(strings.ToLower(arg), strings.ToLower("http")) {
			newArgs = append(newArgs, arg)
		} else {
			absPath, err := filepath.Abs(arg)
			if err != nil {
				log.Fatalf("%s: Error getting absolute path [%v]\n",
					color.RedString(constants.FATAL_NORMAL_CASE), err)
			} else {
				dir := filepath.Dir(absPath)
				fileName := filepath.Base(absPath)
				newArgs = append(newArgs, filepath.Join(dir, fileName))
			}
		}
	}

	return newArgs
}

func waitForKeyPress() {
	if err := keyboard.Open(); err != nil {
        log.Fatal(err)
    }
    defer keyboard.Close()

	fmt.Println("Press ANY key to continue...")
    _, _, err := keyboard.GetSingleKey()
    if err != nil {
		log.Fatalf("%s: %v\n",
			color.RedString(constants.FATAL_NORMAL_CASE), err)
    }
}

