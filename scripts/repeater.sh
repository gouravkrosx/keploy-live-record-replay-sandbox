#!/bin/bash

# Set the duration to 30 minutes (1800 seconds)
START_TIME=$(date +%s)
END_TIME=$(( START_TIME + 1800 ))

echo "Starting repeater script for 30 minutes..."

while [ $(date +%s) -lt $END_TIME ]; do
    echo "Running load test command..."
    python3 scripts/load_test.py --ops 200 --delay 0.1 --k8s
    
    # Calculate elapsed time
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    E_MIN=$((ELAPSED / 60))
    E_SEC=$((ELAPSED % 60))
    
    echo "Time elapsed: ${E_MIN}m ${E_SEC}s"
    
    # Check if the 30 minutes have passed before waiting
    if [ $(date +%s) -lt $END_TIME ]; then
        echo "Waiting 30 seconds before next run..."
        for i in {1..30}; do
             CURRENT_TIME=$(date +%s)
             ELAPSED=$((CURRENT_TIME - START_TIME))
             # Print on the same line to avoid spamming
             echo -ne "\rWaiting... ${i}s/30s | Total Elapsed: ${ELAPSED}s" 
             sleep 1
        done
        echo "" # New line after wait is done
    fi
done

echo "30 minutes have surpassed. Exiting."
