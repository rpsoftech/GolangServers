@echo off
rem run this script as admin

if not exist main.exe (
    echo Build the example before installing by running "go build"
    goto :exit
)

sc create rps-bg-service binpath= "%CD%\main.exe" start= auto DisplayName= "RPS Backgroung Service"
sc description rps-bg-service "Run Services in Backgroung by RPS"
net start rps-bg-service
sc query rps-bg-service

echo Check rps-bg-service.log

:exit