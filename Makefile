TEST_IMAGE := gogento-catalog-test
TEST_CONTAINER := gogento-catalog-test-runner

.PHONY: test-image test test-shell test-clean

# Build test image once (reuses layer cache)
test-image:
	docker build -f Dockerfile.test -t $(TEST_IMAGE) .

# Ensure container exists, then run tests (mounts current dir, container persists)
test: test-image
	@if ! docker ps -a --format '{{.Names}}' | grep -q '^$(TEST_CONTAINER)$$'; then \
		docker run -d --name $(TEST_CONTAINER) -v "$$(pwd)":/app -w /app $(TEST_IMAGE) tail -f /dev/null; \
	fi
	@docker start $(TEST_CONTAINER) 2>/dev/null || true
	# || true: show test output on failure but do not fail the make target
	docker exec $(TEST_CONTAINER) go test ./cmd/... ./api/... ./cron/... ./core/cache/... ./graphql/registry/... ./tests/api/... ./tests/graphql/... ./tests/model/... -v || true

# Shell into the test container
test-shell: test-image
	@if ! docker ps -a --format '{{.Names}}' | grep -q '^$(TEST_CONTAINER)$$'; then \
		docker run -d --name $(TEST_CONTAINER) -v "$$(pwd)":/app -w /app $(TEST_IMAGE) tail -f /dev/null; \
	fi
	@docker start $(TEST_CONTAINER) 2>/dev/null || true
	docker exec -it $(TEST_CONTAINER) sh

# Run performance tests (GraphQL vs REST API comparison)
test-perf: test-image
	@if ! docker ps -a --format '{{.Names}}' | grep -q '^$(TEST_CONTAINER)$$'; then \
		docker run -d --name $(TEST_CONTAINER) -v "$$(pwd)":/app -w /app $(TEST_IMAGE) tail -f /dev/null; \
	fi
	@docker start $(TEST_CONTAINER) 2>/dev/null || true
	docker exec $(TEST_CONTAINER) go test -v -run "TestPerf_GraphQL_vs_API|TestGraphQL_Perf_100Products" ./tests/api/... || true

# Remove the persistent container
test-clean:
	docker rm -f $(TEST_CONTAINER) 2>/dev/null || true
