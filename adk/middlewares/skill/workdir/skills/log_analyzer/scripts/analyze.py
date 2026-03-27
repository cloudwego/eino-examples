import sys
import os

def analyze_log(file_path):
    if not os.path.exists(file_path):
        print(f"Error: File '{file_path}' not found.")
        return

    error_count = 0
    warning_count = 0
    error_lines = []
    warning_lines = []

    try:
        with open(file_path, 'r') as f:
            for i, line in enumerate(f, 1):
                content = line.strip()
                if "ERROR" in content:
                    error_count += 1
                    error_lines.append(f"Line {i}: {content}")
                elif "WARNING" in content:
                    warning_count += 1
                    warning_lines.append(f"Line {i}: {content}")
        
        print(f"Analysis Result for {file_path}:")
        print(f"Total Errors: {error_count}")
        print(f"Total Warnings: {warning_count}")
        
        if error_count > 0:
            print("\nError Details:")
            for line in error_lines:
                print(line)
                
        if warning_count > 0:
            print("\nWarning Details:")
            for line in warning_lines:
                print(line)

    except Exception as e:
        print(f"An error occurred while reading the file: {e}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 analyze.py <log_file_path>")
    else:
        analyze_log(sys.argv[1])
