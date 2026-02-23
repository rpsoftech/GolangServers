@echo off
rem run this script as admin

net stop rps-bg-service
sc delete rps-bg-service