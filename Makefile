DEBUG_FLAG = $(if $(DEBUG), -debug)

VERSION_GO = version.go

_NAME      = $(shell grep -o 'AppName string = "[^"]*"' $(VERSION_GO)  | cut -d '"' -f2)
_VERSION   = $(shell grep -oE 'Version string = "[0-9]+\.[0-9]+\.[0-9]+"' $(VERSION_GO) | cut -d '"' -f2)

.PHONY: build
build:
	docker build --build-arg VERSION=$(_VERSION) -t $(_NAME):$(_VERSION) .
	docker tag $(_NAME):$(_VERSION) $(_NAME):latest

.PHONY: ghpkg
ghpkg: 
	docker tag $(_NAME):$(_VERSION) docker.pkg.github.com/octu0/revproxy/$(_NAME):$(_VERSION)
	docker push docker.pkg.github.com/octu0/revproxy/$(_NAME):$(_VERSION)

.PHONY: pkg
pkg:
	mkdir -p "$(PWD)/pkg"
	$(eval binpath_linux := "$(PWD)/pkg/$(_NAME)_linux_amd64-$(_VERSION)")
	$(eval binpath_darwin := "$(PWD)/pkg/$(_NAME)_darwin_amd64-$(_VERSION)")
	$(eval cid := $(shell docker create $(_NAME):$(_VERSION)))
	docker cp $(cid):/app/$(_NAME) "$(binpath_linux)"
	docker cp $(cid):/app/$(_NAME)_darwin "$(binpath_darwin)"
	docker rm -v $(cid)
	zip -j "$(binpath_linux).zip" "$(binpath_linux)"
	zip -j "$(binpath_darwin).zip" "$(binpath_darwin)"

.PHONY: ghauth
ghauth:
	cat ~/.GH_TOKEN | docker login -u --password-stdin docker.pkg.github.com
