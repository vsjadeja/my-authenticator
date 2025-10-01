# Makefile for My Authenticator (TOTP GUI Application)

# Variables
APP_NAME = my-authenticator
ICON = icon.png
MAIN_FILE = main.go
BUILD_DIR = build
BINARY = $(BUILD_DIR)/$(APP_NAME)

# Go build flags for Fyne GUI with system libraries (from build.sh)
export CGO_LDFLAGS = -L/usr/lib/x86_64-linux-gnu
export CGO_CPPFLAGS = -I/usr/include

# Create build directory
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# Build the application
.PHONY: build
build: $(BUILD_DIR)
	@echo "🔨 Building $(APP_NAME)..."
	go build -o $(BINARY) $(MAIN_FILE)
	@echo "✅ Build complete: $(BINARY)"

# Download and tidy dependencies
.PHONY: deps
deps:
	@echo "📦 Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies updated"
	@echo "🧹 Downloading Fyne CLI..."
	go install fyne.io/fyne/v2/cmd/fyne@latest
	@echo "✅ Fyne CLI installed"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(APP_NAME)
	@echo "✅ Clean complete"

# Build Debian package
.PHONY: deb
deb: build
	@echo "📦 Creating Debian package..."
	@mkdir -p $(BUILD_DIR)/deb/DEBIAN
	@mkdir -p $(BUILD_DIR)/deb/usr/bin
	@mkdir -p $(BUILD_DIR)/deb/usr/share/applications
	@mkdir -p $(BUILD_DIR)/deb/usr/share/pixmaps
	@echo "Package: $(APP_NAME)" > $(BUILD_DIR)/deb/DEBIAN/control
	@echo "Version: 1.0.0" >> $(BUILD_DIR)/deb/DEBIAN/control
	@echo "Section: utils" >> $(BUILD_DIR)/deb/DEBIAN/control
	@echo "Priority: optional" >> $(BUILD_DIR)/deb/DEBIAN/control
	@echo "Architecture: amd64" >> $(BUILD_DIR)/deb/DEBIAN/control
	@echo "Maintainer: Your Name <your.email@example.com>" >> $(BUILD_DIR)/deb/DEBIAN/control
	@echo "Description: TOTP Authenticator GUI Application" >> $(BUILD_DIR)/deb/DEBIAN/control
	@echo " A secure Time-based One-Time Password (TOTP) authenticator" >> $(BUILD_DIR)/deb/DEBIAN/control
	@echo " similar to Google Authenticator with encrypted storage." >> $(BUILD_DIR)/deb/DEBIAN/control
	@cp $(BINARY) $(BUILD_DIR)/deb/usr/bin/
	@cp $(ICON) $(BUILD_DIR)/deb/usr/share/pixmaps/$(APP_NAME).png
	@echo "[Desktop Entry]" > $(BUILD_DIR)/deb/usr/share/applications/$(APP_NAME).desktop
	@echo "Name=My Authenticator" >> $(BUILD_DIR)/deb/usr/share/applications/$(APP_NAME).desktop
	@echo "Comment=TOTP Authenticator GUI Application" >> $(BUILD_DIR)/deb/usr/share/applications/$(APP_NAME).desktop
	@echo "Exec=/usr/bin/$(APP_NAME)" >> $(BUILD_DIR)/deb/usr/share/applications/$(APP_NAME).desktop
	@echo "Icon=/usr/share/pixmaps/$(APP_NAME).png" >> $(BUILD_DIR)/deb/usr/share/applications/$(APP_NAME).desktop
	@echo "Terminal=false" >> $(BUILD_DIR)/deb/usr/share/applications/$(APP_NAME).desktop
	@echo "Type=Application" >> $(BUILD_DIR)/deb/usr/share/applications/$(APP_NAME).desktop
	@echo "Categories=Utility;Security;" >> $(BUILD_DIR)/deb/usr/share/applications/$(APP_NAME).desktop
	@chmod 755 $(BUILD_DIR)/deb/usr/bin/$(APP_NAME)
	@dpkg-deb --build $(BUILD_DIR)/deb $(BUILD_DIR)/$(APP_NAME).deb
	@echo "✅ Debian package created: $(BUILD_DIR)/$(APP_NAME).deb"

.PHONY: run
run: clean build
	@echo "🚀 Running $(APP_NAME)..."
	@$(BINARY)