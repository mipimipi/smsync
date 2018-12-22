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

# set project VERSION if VERSION hasn't been passed from command line
ifndef $(value VERSION)
	VERSION=$(cat ./VERSION)
endif

# setup the -ldflags option for go build
LDFLAGS=-ldflags "-X main.Version=$(value VERSION)"

# build all executables
all:
	go build $(LDFLAGS) ./cmd/...

.PHONY: all clean install lint release

lint:
	gometalinter \
		--enable=goimports \
		--enable=misspell \
		--enable=nakedret \
		--enable=unparam \
		--disable=gocyclo \
		--vendor \
		--deadline=30s \
		./...

# move all executables to /usr/bin 
install:
	for CMD in `ls cmd`; do \
		install -Dm755 $$CMD $(DESTDIR)/usr/bin/$$CMD; \
	done

# create a new release tag
release:
	git tag -a $(value VERSION) -m "Release $(value VERSION)"
	git push origin $(value VERSION)	

# remove build results
clean:
	for CMD in `ls cmd`; do \
		rm -f ./$$CMD; \
	done