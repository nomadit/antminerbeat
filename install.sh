#!/bin/bash
cd /home/miner/antminerbeat/beat
sudo rm -f antminerbeat-real.yml
ln -s antminerbeat-real.yml antminerbeat.yml
sudo chown root: *yml
sudo systemctl enable /home/miner/antminerbeat/beat/antminerbeat.service
sudo systemctl enable /home/miner/antminerbeat/scanner/scanner.service
cd /home/miner/antminerbeat
