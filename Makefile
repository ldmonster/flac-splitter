.PHONY: build run clean install-deps license-headers help

# Build the FLAC splitter
build:
	@echo "Building FLAC splitter..."
	go build ./cmd/flac-splitter
	@echo "Build complete! Binary: flac-splitter"

# Build and run
run: build
	@./flac-splitter

# Clean build artifacts and output
clean:
	@echo "Cleaning up..."
	rm -f flac-splitter
	rm -rf split/
	@echo "Clean complete!"

# Clean only the split output directory
clean-output:
	@echo "Removing split directory..."
	rm -rf split/
	@echo "Output directory cleaned!"

# Install dependencies (Debian/Ubuntu)
install-deps:
	@echo "Installing dependencies..."
	@if command -v apt-get &> /dev/null; then \
		sudo apt-get update && \
		sudo apt-get install -y shntool flac ffmpeg; \
	elif command -v brew &> /dev/null; then \
		brew install shntool flac ffmpeg; \
	elif command -v pacman &> /dev/null; then \
		sudo pacman -S shntool flac ffmpeg; \
	else \
		echo "Package manager not recognized. Please install shntool+flac or ffmpeg manually."; \
	fi
	@echo "Dependencies installed!"

# Add license headers to all Go files
license-headers:
	@echo "Adding license headers to Go files..."
	@for file in $$(find . -name '*.go' -type f); do \
		if ! grep -q "Licensed under the Apache License" "$$file"; then \
			tmpfile=$$(mktemp); \
			echo "// Copyright 2026 ldmonster" > "$$tmpfile"; \
			echo "//" >> "$$tmpfile"; \
			echo "// Licensed under the Apache License, Version 2.0 (the \"License\");" >> "$$tmpfile"; \
			echo "// you may not use this file except in compliance with the License." >> "$$tmpfile"; \
			echo "// You may obtain a copy of the License at" >> "$$tmpfile"; \
			echo "//" >> "$$tmpfile"; \
			echo "//     http://www.apache.org/licenses/LICENSE-2.0" >> "$$tmpfile"; \
			echo "//" >> "$$tmpfile"; \
			echo "// Unless required by applicable law or agreed to in writing, software" >> "$$tmpfile"; \
			echo "// distributed under the License is distributed on an \"AS IS\" BASIS," >> "$$tmpfile"; \
			echo "// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied." >> "$$tmpfile"; \
			echo "// See the License for the specific language governing permissions and" >> "$$tmpfile"; \
			echo "// limitations under the License." >> "$$tmpfile"; \
			echo "" >> "$$tmpfile"; \
			cat "$$file" >> "$$tmpfile"; \
			mv "$$tmpfile" "$$file"; \
			echo "  Added header to $$file"; \
		fi; \
	done
	@echo "License headers complete!"

# Help
help:
	@echo "FLAC Splitter Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make build         - Build the FLAC splitter"
	@echo "  make run           - Build and run the FLAC splitter in current directory"
	@echo "  make clean         - Remove build artifacts and output directory"
	@echo "  make clean-output  - Remove only the split output directory"
	@echo "  make install-deps  - Install required dependencies (shntool/ffmpeg)"
	@echo "  make license-headers - Add Apache 2.0 license headers to all Go files"
	@echo "  make help          - Show this help message"
