# Makefile for sonoscli-gui

APP_NAME=sonoscli-gui
BINARY_NAME=sonoscli-gui

.PHONY: all build package-mac package-mac-pkg package-linux package-windows clean deps

all: build

deps:
	@chmod +x scripts/install_deps.sh
	./scripts/install_deps.sh
	go install fyne.io/fyne/v2/cmd/fyne@latest
	go mod download


build:
	go build -o $(BINARY_NAME) .

package-mac:
	@if [ -f "Icon.png" ]; then \
		fyne package -os darwin -icon Icon.png; \
	else \
		fyne package -os darwin; \
	fi

package-mac-pkg: package-mac
	pkgbuild --component $(APP_NAME).app --install-location /Applications --identifier com.sonoscli.gui $(APP_NAME).pkg

package-linux:
	@if [ -f "Icon.png" ]; then \
		fyne package -os linux -icon Icon.png; \
	else \
		fyne package -os linux; \
	fi

package-windows:
	@if [ -f "Icon.png" ]; then \
		fyne package -os windows -icon Icon.png; \
	else \
		fyne package -os windows; \
	fi

clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -rf $(APP_NAME).app
	rm -f $(APP_NAME).tar.gz
	rm -f $(APP_NAME).pkg
