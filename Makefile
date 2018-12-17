# Copyright (C) 2018 Michael Picht
#
# This file is part of smsync (Smart Music Sync).
#
# smsync is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# smsync is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with smsync. If not, see <http://www.gnu.org/licenses/>.

# use bash
SHELL=/bin/bash

# set project VERSION
VERSION=$(cat ./VERSION)

# setup the -ldflags option for go build
LDFLAGS=-ldflags "-X main.Version=$(value VERSION)"

# build all executables
all:
	dep ensure
	go build $(LDFLAGS) ./cmd/...

$(GOMETALINTER):
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install &> /dev/null

.PHONY: lint

lint: $(GOMETALINTER)
	gometalinter ./... --vendor

# move all executables to /usr/bin and 
install:
	for CMD in `ls cmd`; do \
		install -Dm755 $$CMD $(DESTDIR)/usr/bin/$$CMD; \
		rm -f ./$$CMD; \
	done

# create a new release tag
#release:
#	git tag -a $(VERSION) -m "Release $(VERSION)" || true
#	git push origin $(VERSION)	