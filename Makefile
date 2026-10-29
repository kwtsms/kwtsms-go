.PHONY: test test-race cover clean lint release

# Run unit tests
test:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -race ./...

# Run tests with coverage report
cover:
	go test -coverprofile=coverage.txt ./...
	go tool cover -func=coverage.txt
	@echo ""
	@echo "HTML report: go tool cover -html=coverage.txt -o coverage.html"

# Run golangci-lint (falls back to go vet)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, falling back to go vet"; \
		go vet ./...; \
	fi

# Clean build artifacts
clean:
	rm -f coverage.txt coverage.html

# ── Release ──────────────────────────────────────────────────────────
#
# Usage: make release VERSION=0.4.0
#
# This will:
#   1. Verify working tree is clean
#   2. Run tests
#   3. Bump version in kwtsms.go
#   4. Update CHANGELOG.md ([Unreleased] → [VERSION])
#   5. Commit, tag, push
#   6. CI + release workflow handle the rest
#
release:
ifndef VERSION
	$(error VERSION is required. Usage: make release VERSION=0.4.0)
endif
	@echo "=== Releasing v$(VERSION) ==="
	@# 1. Clean working tree
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "ERROR: Working tree is not clean. Commit or stash changes first."; \
		exit 1; \
	fi
	@# 2. Verify [Unreleased] section exists and has content
	@if ! grep -q '## \[Unreleased\]' CHANGELOG.md; then \
		echo "ERROR: No [Unreleased] section in CHANGELOG.md"; \
		exit 1; \
	fi
	@CONTENT=$$(awk '/^## \[Unreleased\]/{found=1; next} /^## \[/{exit} found{print}' CHANGELOG.md | grep -v '^$$' | head -1); \
	if [ -z "$$CONTENT" ]; then \
		echo "ERROR: [Unreleased] section in CHANGELOG.md is empty"; \
		exit 1; \
	fi
	@# 3. Run tests
	@echo "--- Running tests ---"
	go test -race ./...
	@echo ""
	@# 4. Bump version in kwtsms.go
	@echo "--- Bumping version to $(VERSION) ---"
	sed -i 's/const Version = ".*"/const Version = "$(VERSION)"/' kwtsms.go
	@# 5. Update CHANGELOG.md
	@echo "--- Updating CHANGELOG.md ---"
	@TODAY=$$(date +%Y-%m-%d); \
	sed -i "s/## \[Unreleased\]/## [Unreleased]\n\n## [$(VERSION)] - $$TODAY/" CHANGELOG.md
	@# Add release link before first existing link
	@PREV=$$(grep -oP '^\[\K[0-9]+\.[0-9]+\.[0-9]+' CHANGELOG.md | head -2 | tail -1); \
	sed -i "/^\[$$PREV\]:/i [$(VERSION)]: https://github.com/kwtsms/kwtsms-go/releases/tag/v$(VERSION)" CHANGELOG.md
	@# 6. Commit, tag, push
	@echo "--- Committing and tagging ---"
	git add kwtsms.go CHANGELOG.md
	git commit -m "Release v$(VERSION)"
	git tag "v$(VERSION)"
	@echo "--- Pushing ---"
	git push origin main "v$(VERSION)"
	@echo ""
	@echo "=== v$(VERSION) released! ==="
	@echo "  GitHub Release: https://github.com/kwtsms/kwtsms-go/releases/tag/v$(VERSION)"
	@echo "  pkg.go.dev:     https://pkg.go.dev/github.com/kwtsms/kwtsms-go@v$(VERSION)"
