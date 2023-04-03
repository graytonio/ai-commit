build:
	go build -o git-ai-commit

install: build
	sudo mv git-ai-commit /usr/local/bin/git-ai-commit
