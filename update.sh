#!/bin/bash
cd /home/miner/antminerbeat
rm -f antminerbeat.tar.gz
wget https://bitbucket.org/acciolabs/sw/downloads/antminerbeat.tar.gz
cp serial_key backup.serial_key
tar xvzf antminerbeat.tar.gz
cp backup.serial_key serial_key
cd /home/miner/antminerbeat/beat
rm -f antminerbeat.yml
ln -s antminerbeat-real.yml antminerbeat.yml
sudo chown root: *yml
cd /home/miner/antminerbeat
