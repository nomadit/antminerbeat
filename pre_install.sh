#!/bin/bash
mkdir /home/miner/antminerbeat
cd /home/miner/antminerbeat
wget https://bitbucket.org/acciolabs/sw/downloads/antminerbeat.tar.gz
tar xvzf antminerbeat.tar.gz
cd /home/miner/antminerbeat/beat
ln -s antminerbeat-real.yml antminerbeat.yml
sudo chown root: *yml
sudo systemctl enable /home/miner/antminerbeat/beat/antminerbeat.service
sudo systemctl enable /home/miner/antminerbeat/scanner/scanner.service
