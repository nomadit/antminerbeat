#!/bin/bash
sudo service antminerbeat stop
sudo service scanner stop
sudo systemctl disable /home/miner/antminerbeat/beat/antminerbeat.service
sudo systemctl disable /home/miner/antminerbeat/scanner/scanner.service
