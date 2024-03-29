WHAT := kubeversion-exporter

BRANCH          ?= $(shell git rev-parse --abbrev-ref HEAD)
BUILDTIME       ?= $(shell date '+%Y%m%d-%H:%M:%S')
BUILDUSER       ?= $(shell id -un)
PWD             ?= $(shell pwd)
REPO            ?= github.com/ricoberger/kubeversion-exporter
REVISION        ?= $(shell git rev-parse HEAD)
VERSION         ?= $(shell git describe --tags)

.PHONY: build build-darwin-amd64 build-linux-amd64 build-windows-amd64 clean docker-build docker-publish release release-major release-minor release-patch

build:
	for target in $(WHAT); do \
		go build -ldflags "-X ${REPO}/pkg/version.Version=${VERSION} \
			-X ${REPO}/pkg/version.Revision=${REVISION} \
			-X ${REPO}/pkg/version.Branch=${BRANCH} \
			-X ${REPO}/pkg/version.BuildUser=${BUILDUSER} \
			-X ${REPO}/pkg/version.BuildDate=${BUILDTIME}" \
			-o ./bin/$$target ./cmd/$$target; \
	done

build-darwin-amd64:
	for target in $(WHAT); do \
		CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -a -installsuffix cgo -ldflags "-X ${REPO}/pkg/version.Version=${VERSION} \
			-X ${REPO}/pkg/version.Revision=${REVISION} \
			-X ${REPO}/pkg/version.Branch=${BRANCH} \
			-X ${REPO}/pkg/version.BuildUser=${BUILDUSER} \
			-X ${REPO}/pkg/version.BuildDate=${BUILDTIME}" \
			-o ./bin/$$target-darwin-amd64 ./cmd/$$target; \
	done

build-linux-amd64:
	for target in $(WHAT); do \
		CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -a -installsuffix cgo -ldflags "-X ${REPO}/pkg/version.Version=${VERSION} \
			-X ${REPO}/pkg/version.Revision=${REVISION} \
			-X ${REPO}/pkg/version.Branch=${BRANCH} \
			-X ${REPO}/pkg/version.BuildUser=${BUILDUSER} \
			-X ${REPO}/pkg/version.BuildDate=${BUILDTIME}" \
			-o ./bin/$$target-linux-amd64 ./cmd/$$target; \
	done

build-windows-amd64:
	for target in $(WHAT); do \
		CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -a -installsuffix cgo -ldflags "-X ${REPO}/pkg/version.Version=${VERSION} \
			-X ${REPO}/pkg/version.Revision=${REVISION} \
			-X ${REPO}/pkg/version.Branch=${BRANCH} \
			-X ${REPO}/pkg/version.BuildUser=${BUILDUSER} \
			-X ${REPO}/pkg/version.BuildDate=${BUILDTIME}" \
			-o ./bin/$$target-windows-amd64.exe ./cmd/$$target; \
	done

clean:
	rm -rf ./bin

docker-build: build-linux-amd64
	for target in $(WHAT); do \
		docker build -f cmd/$$target/Dockerfile -t "$$target:${VERSION}" --build-arg REVISION=${REVISION} --build-arg VERSION=${VERSION} .; \
	done

docker-publish:
	for target in $(WHAT); do \
		docker tag $$target:${VERSION} ricoberger/$$target:${VERSION}; \
		docker tag $$target:${VERSION} docker.pkg.github.com/ricoberger/kubeversion-exporter/$$target:${VERSION}; \
		docker push ricoberger/$$target:${VERSION}; \
		docker push docker.pkg.github.com/ricoberger/kubeversion-exporter/$$target:${VERSION}; \
	done

release: clean docker-build docker-publish

release-major:
	$(eval MAJORVERSION=$(shell git describe --tags --abbrev=0 | sed s/v// | awk -F. '{print $$1+1".0.0"}'))
	git checkout master
	git pull
	git tag -a $(MAJORVERSION) -m 'release $(MAJORVERSION)'
	git push origin --tags

release-minor:
	$(eval MINORVERSION=$(shell git describe --tags --abbrev=0 | sed s/v// | awk -F. '{print $$1"."$$2+1".0"}'))
	git checkout master
	git pull
	git tag -a $(MINORVERSION) -m 'release $(MINORVERSION)'
	git push origin --tags

release-patch:
	$(eval PATCHVERSION=$(shell git describe --tags --abbrev=0 | sed s/v// | awk -F. '{print $$1"."$$2"."$$3+1}'))
	git checkout master
	git pull
	git tag -a $(PATCHVERSION) -m 'release $(PATCHVERSION)'
	git push origin --tags
