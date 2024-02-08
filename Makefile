lint::
	cd "pkg" && golangci-lint run -c ../.golangci.yml --timeout 10m
	cd "cmd" && golangci-lint run -c ../.golangci.yml --timeout 10m
