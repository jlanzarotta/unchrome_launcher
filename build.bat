@echo off
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
set BUILD_VERSION=1.0.4
set BUILD_DATETIME="%date:~10,4%-%date:~4,2%-%date:~7,2%T%time: =0%"
go build -ldflags="-s -w -X 'ungoogled_launcher/cmd.BuildDateTime=%BUILD_DATETIME%' -X 'ungoogled_launcher/cmd.BuildVersion=%BUILD_VERSION%'" -work -a -v -o ungoogled_launcher.exe main.go
