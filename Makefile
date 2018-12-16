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

# project directory
PROJECT=github.com/mipimipi/smsync

# set project VERSION if VERSION hasn't been passed from command line
ifndef $(VERSION)
	VERSION=3.0.2
endif

# use bash
SHELL=/bin/bash

# setup the -ldflags option for go build
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

all:
	# build all executables
	for CMD in `ls cmd`; do \
		go build $(LDFLAGS) ./cmd/$$CMD; \
	done

$(GOMETALINTER):
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install &> /dev/null

.PHONY: lint
lint: $(GOMETALINTER)
	gometalinter ./... --vendor

install:
	# copy all executables to /usr/bin
	for CMD in `ls cmd`; do \
		install -Dm755 $$CMD $(DESTDIR)/usr/bin/$$CMD; \
	done

