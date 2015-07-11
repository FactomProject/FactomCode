#!/usr/bin/env bash
#
# To aid with server debugging, save stdout & stderr to the log file.
# 
# Please run as:
#   runfactomd.sh &
#
# mkdir ~/logs/ 2> /dev/null
# factomd --debuglevel=debug 2>&1 | tee -a ~/logs/factomd_${RANDOM}_${RANDOM}.log
factomd >> ~/.factom/logs/factom-d.log
