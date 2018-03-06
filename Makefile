# Copyright (C) 2018 Michael Picht
#
# This file is part of smsync (Smart Music Sync).
#
# smsync is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# gool is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with smsync. If not, see <http://www.gnu.org/licenses/>.

# This is how we want to name the binary output
BINARY=smsync

# set VERSION if VERSION hasn't been passed from command line
ifndef $(VERSION)
	VERSION=1.3.1
endif

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

all:
	go build ${LDFLAGS} -o ${BINARY}

$(GOMETALINTER):
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install &> /dev/null

.PHONY: lint
lint: $(GOMETALINTER)
	gometalinter ./... --vendor

install:
	install -Dm755 ${BINARY} $(DESTDIR)/usr/bin/${BINARY}