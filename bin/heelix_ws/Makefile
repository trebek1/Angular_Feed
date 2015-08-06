all : install


clean :
	@echo ">>> Cleaning and initializing project <<<"
	go clean
	go get github.com/stretchr/testify


test : clean
	@echo ">>> Running unit tests <<<"
	go clean
	go test ./


test-cover : clean
	@echo ">>> Running code coverage check <<<"
	go clean
	go test ./ -cover


coverage-report : 
	@echo ">>> Running code coverage and generating HTML report <<<"
	go clean
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out


install : test
	@echo ">>> Building and installing heelix_ws <<<"
	go install
	@echo ">>> heelix_ws installed successfully! <<<"
	@echo ""
