#!/usr/bin/env python3
import os
import subprocess
import sys

def get_text_size(filepath):
    """Get the .text section size using 'size -A' command"""
    try:
        output = subprocess.check_output(
            ['size', '-A', filepath],
            stderr=subprocess.STDOUT,
            universal_newlines=True
        )
    except (subprocess.CalledProcessError, FileNotFoundError):
        return None

    for line in output.splitlines():
        parts = line.split()
        if len(parts) >= 2 and parts[0] == '.text':
            try:
                return int(parts[1])
            except ValueError:
                continue
    return None

def main():
    if len(sys.argv) != 3:
        print("Usage: ./compare_text_size.py <dir1> <dir2>")
        print("Compares .text section sizes of executables in two directories")
        sys.exit(1)

    dir1 = sys.argv[1]
    dir2 = sys.argv[2]

    # Verify directories exist
    if not os.path.isdir(dir1):
        print(f"Error: '{dir1}' is not a directory")
        sys.exit(1)
    if not os.path.isdir(dir2):
        print(f"Error: '{dir2}' is not a directory")
        sys.exit(1)

    # Find executable files in first directory
    executables = []
    for fname in os.listdir(dir1):
        path = os.path.join(dir1, fname)
        if os.path.isfile(path) and os.access(path, os.X_OK):
            executables.append(fname)

    # Compare each executable with counterpart in second directory
    results = []
    for exe in sorted(executables):
        path1 = os.path.join(dir1, exe)
        path2 = os.path.join(dir2, exe)

        if not os.path.isfile(path2):
            print(f"Warning: '{exe}' not found in second directory", file=sys.stderr)
            continue

        size1 = get_text_size(path1)
        size2 = get_text_size(path2)

        if size1 is None:
            print(f"Warning: Couldn't get .text size for '{path1}'", file=sys.stderr)
            continue
        if size2 is None:
            print(f"Warning: Couldn't get .text size for '{path2}'", file=sys.stderr)
            continue

        # Calculate percentage change
        if size1 == 0:
            pct_change = "N/A (old=0)"
        else:
            change = (size2 - size1) / size1 * 100
            pct_change = f"{change:+.2f}%"

        results.append((exe, size1, size2, pct_change))

    # Print results
    if not results:
        print("No comparable executables found")
        return

    print(f"{'Executable':<20} {'Old .text':>10} {'New .text':>10} {'Change':>10}")
    print("-" * 55)
    for name, old, new, change in results:
        print(f"{name:<20} {old:>10} {new:>10} {change:>10}")

if __name__ == "__main__":
    main()
