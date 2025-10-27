.PHONY: test test-headless

# test runs go test ./... automatically using xvfb-run when available.
test:
	@if command -v xvfb-run >/dev/null 2>&1; then \
		echo "Running go test with xvfb-run"; \
		xvfb-run -a go test ./...; \
	else \
		echo "xvfb-run not found; running go test directly"; \
		go test ./...; \
	fi

# test-headless always uses xvfb-run to provide an X11 display for Ebiten tests.
test-headless:
	xvfb-run -a go test ./...
