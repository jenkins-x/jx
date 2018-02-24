featurePath = $(PWD)
PKGS := $(shell go list ./... | grep -v /vendor)

fmt:
	find ! -path "./vendor/*" -name "*.go" -exec gofmt -s -w {} \;

gometalinter:
	gometalinter -D gotype -D aligncheck --vendor --deadline=600s --dupl-threshold=200 -e '_string' -j 5 ./...

doc-hunt:
	doc-hunt check -e

setup-test-fixtures:
	cd cmd && sh $(featurePath)/features/init.sh
	cd cmd && sh $(featurePath)/features/merge-commits.sh
	cd chyle && sh $(featurePath)/features/init.sh
	cd chyle && sh $(featurePath)/features/merge-commits.sh
	cd chyle/git &&	sh $(featurePath)/features/init.sh
	cd chyle/git && sh $(featurePath)/features/merge-commits.sh

run-tests: setup-test-fixtures
	./test.sh

run-quick-tests: setup-test-fixtures
	go test -v $(PKGS)

test-package:
	go test -race -cover -coverprofile=/tmp/chyle github.com/antham/chyle/$(pkg)
	go tool cover -html=/tmp/chyle -o /tmp/chyle.html
