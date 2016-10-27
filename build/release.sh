#!/bin/bash

VERSION="0.9.0"

BUILD_PATH="$PWD"
#BUILD_PATH="/tmp/dmoni/build"

REPO_PATH="github.com/lizhongz/dmoni"
BIN=dmoni
SNAPSHOT=snapshot
README=README.md

# Check if build dir exists
if [ ! -d "$BUILD_PATH" ]; then
    mkdir -p $BUILD_PATH
fi

cd $BUILD_PATH

# Build DMoni
go build -o $BIN $REPO_PATH 

# Copy snapshot repo
git clone --depth=1 git@gadget.ncsl.nist.gov:lizhong/snapshot.git $SNAPSHOT >/dev/null && rm -rf $SNAPSHOT/.git

# Copy README
cp $GOPATH/src/$REPO_PATH/README.md $README

# Create a tarball
tar --transform 's,^,dmoni/,' -zcvf $BIN-$VERSION.tar.gz $BIN $SNAPSHOT $README >/dev/null
 
# Clean up
rm -rf $BIN $SNAPSHOT $README

cd - >/dev/null
