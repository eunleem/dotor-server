#!/bin/bash
go build -o dotor.exe
sudo systemctl restart dotor
sleep 2
sudo systemctl status dotor -l
