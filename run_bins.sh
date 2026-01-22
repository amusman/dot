#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Function to display usage information
usage() {
    echo "Usage: $0 <num_runs> <bin1> <bin2> ... <binN>"
    echo "  <num_runs> : Number of runs for each binary (positive integer)"
    echo "  <binX>     : Path to binary executable (at least one required)"
    echo '  for example: PREFIX="taskset -c 44-47 " SUFFIX=" -test.run=none -test.bench=CopyFat56 -test.count 1 " run_bins.sh 10 ./orig ./movq'
    exit 1
}

# Check if at least two arguments are provided (num_runs + at least one binary)
if [ "$#" -lt 2 ]; then
    echo "Error: Invalid number of arguments."
    usage
fi

# First argument is the number of runs
NUM_RUNS="$1"
shift  # Remove first argument, leaving only binaries

# Validate that NUM_RUNS is a positive integer
if ! [[ "$NUM_RUNS" =~ ^[1-9][0-9]*$ ]]; then
    echo "Error: <num_runs> must be a positive integer."
    exit 1
fi

# Array to store binary paths
BINARIES=("$@")

# Check that we have at least one binary
if [ ${#BINARIES[@]} -eq 0 ]; then
    echo "Error: At least one binary must be specified."
    usage
fi

# Validate that each binary exists and is executable
for BINARY in "${BINARIES[@]}"; do
    if [ ! -x "$BINARY" ]; then
        echo "Error: '$BINARY' is not an executable file."
        exit 1
    fi
done

# Get the total number of runs for all binaries
TOTAL_RUNS=$((NUM_RUNS * ${#BINARIES[@]}))

# Loop through total runs
for ((i=1; i<=TOTAL_RUNS; i++))
do
    # Calculate which binary to run (alternating through the list)
    # (i-1) % number_of_binaries gives us 0,1,2,0,1,2,...
    binary_index=$(( (i - 1) % ${#BINARIES[@]} ))
    BINARY="${BINARIES[binary_index]}"

    echo "Run $i: Executing $PREFIX $BINARY $SUFFIX"
    $PREFIX $BINARY $SUFFIX >> "${BINARY}.out"

    # Sleep for 5 seconds between runs, except after the last run
    if (( i < TOTAL_RUNS )); then
        sleep 5
    fi
done

echo "All done"
