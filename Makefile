.PHONY: vespyr test

vespyr:
	go install github.com/DavidHuie/vespyr/cmd/vespyr

test:
	go test github.com/DavidHuie/vespyr/pkg/...

test_long:
	go test -v -race -cover github.com/DavidHuie/vespyr/pkg/...

test_short:
	go test -v -short github.com/DavidHuie/vespyr/pkg/...
